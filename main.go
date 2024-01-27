package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/fatih/color"
	"github.com/samber/lo"
	"go-tunnel/auth/basic"
	"go-tunnel/auth/digest"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type Config struct {
	Remote            string   `yaml:"remote"`
	PKPath            string   `yaml:"pkpath"`
	Tunnels           []string `yaml:"tunnels"`
	AuthMethod        string   `yaml:"auth-method"`
	AuthUser          string   `yaml:"auth-user"`
	AuthPass          string   `yaml:"auth-pass"`
	Ssl               bool     `yaml:"ssl"`
	AllowedCerts      []string `yaml:"allowed-certs"`
	MaxConnections    int32    `yaml:"max-connections"`
	MaxHttpHeaderSize int32    `yaml:"max-http-header-size"`
	LogTraffic        bool     `yaml:"log-traffic"`
}

type HttpAuthenticator interface {
	Check(header, user, pass string) bool
	UnauthorizedHeader() string
}

var tlsConfig tls.Config
var httpAuthenticator HttpAuthenticator
var activeConnections int32

func main() {
	log.Println("Starting...")
	config := readConfig("config.yaml")
	if config.AuthMethod == "basic" {
		httpAuthenticator = basic.New()
		log.Println("Authentication method: Basic")
	} else {
		httpAuthenticator = digest.New()
		log.Println("Authentication method: Digest")
	}

	// init default config values
	if config.MaxConnections == 0 {
		config.MaxConnections = 100
	}
	if config.MaxHttpHeaderSize == 0 {
		config.MaxHttpHeaderSize = 8192
	}

	// SSL enabled
	if config.Ssl {
		log.Println("SSL is enabled")
		cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
		if err != nil {
			log.Fatalf("server: loadkeys: %s", err)
		}

		caCert, err := os.ReadFile("ca.crt")
		if err != nil {
			log.Fatalf("Failed to read CA certificate: %s", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig = tls.Config{
			ClientCAs:    caCertPool,
			Certificates: []tls.Certificate{cert},
			// ClientAuth:   tls.VerifyClientCertIfGiven, // we want to show message if no certificates given
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
	}

	if config.LogTraffic {
		go trafficLogCleanupLoop()
	}

	// establish ssh connection loop
	for {
		log.Println("establishing SSH connection to", config.Remote, "...")
		var hostport = strings.Split(config.Remote, "@")[1]
		if !strings.Contains(hostport, ":") {
			hostport += ":22"
		}
		client, err := ssh.Dial("tcp", hostport, prepareSshConfig(config))
		if err != nil {
			log.Printf("Failed to dial %s: %s", hostport, err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Println("SSH connection established successfully")

		go keepAliveLoop(client, time.Second*20)

		for _, tunnel := range config.Tunnels {
			parts := strings.Split(tunnel, ":")
			if len(parts) != 4 {
				log.Fatalf("Invalid tunnel config: %s", tunnel)
			}
			addrRem := fmt.Sprintf("%s:%s", parts[0], parts[1])
			addrLoc := fmt.Sprintf("%s:%s", parts[2], parts[3])
			log.Println("starting tunnel between remote port", addrRem, "<--> and local", addrLoc, "...")
			go startTunnel(client, addrRem, addrLoc, config)
		}
		err = client.Conn.Wait() // Blocks execution here! Until connection is closed
		log.Println("\n\n\n***** Connection broken ***** : " + err.Error() + ". Reconnect ...")
		time.Sleep(time.Second * 5)
	}

}

// starts tunnel using given SSH connection
// (similar to `ssh -R ...` command)
// and then starting to accept new connections on that port (addrRemote)
// and then handling those connections with handleConnection(...)
func startTunnel(client *ssh.Client, addrRemote, addrLocal string, cfg Config) {

	var remoteListener net.Listener
	for {
		var err error
		remoteListener, err = client.Listen("tcp", addrRemote)
		if err == nil {
			log.Println("Listen remote:", addrRemote)
			break
		}
		if err != nil && strings.Contains(err.Error(), "forward request denied by peer") {
			log.Printf("Failed to listen on remote %s: %v. Retry in 10 sec..", addrRemote, err)
			time.Sleep(time.Second * 10)
			continue
		}
		if err != nil {
			log.Printf("Failed to listen on remote %s: %v. Exit", addrRemote, err)
			client.Close()
			return
		}
	}

	// infinite loop to accept new connections
	for {
		remoteConn, err := remoteListener.Accept()

		if err != nil {
			log.Printf("Failed to accept remote connection: %v", err)
			if err == io.EOF {
				//remoteConn.Close()
				//client.Close()
				return
			}
			continue
		}

		// check connections limit
		if atomic.LoadInt32(&activeConnections) >= cfg.MaxConnections {
			log.Println("MAX connections limit reached. Close connection")
			remoteConn.Close()
			continue
		}
		incConnectionsCount(1) // increment activeConnections

		localConn, err := net.Dial("tcp", addrLocal)
		if err != nil {
			log.Printf("Failed to connect to local address: %v", err)
			incConnectionsCount(-1) // decrement connection counter
			continue
		}

		log.Println()
		log.Println()
		log.Println()
		log.Println("            ACCEPTED NEW CONNECTION")
		log.Println()
		log.Printf(" Local: %-20s -> %-20s", localConn.LocalAddr(), localConn.RemoteAddr())
		log.Printf("Remote: %-20s -> %-20s", remoteConn.LocalAddr(), remoteConn.RemoteAddr())

		if cfg.Ssl {
			// wrap the connection with TLS
			tlsConn := tls.Server(remoteConn, &tlsConfig)

			if len(cfg.AllowedCerts) > 0 { // if allowed certs are configured
				if _, err := checkClientCertificate(tlsConn, cfg.AllowedCerts); err != nil {
					msg := "check cert error: " + err.Error()
					log.Println(msg)
					tlsConn.Write([]byte("HTTP/1.1 403 Forbidden\r\n"))
					tlsConn.Write([]byte("Content-Type: text/plain\r\n"))
					tlsConn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(msg))))
					tlsConn.Write([]byte(msg))
					tlsConn.Close()
					incConnectionsCount(-1)
					continue
				}
			}

			// replace it with tls
			remoteConn = tlsConn
		}

		go func() {
			handleConnection(remoteConn, localConn, cfg)
			incConnectionsCount(-1) // decrement connection counter
		}()
	}

}

// handle massages/data between remote and local connection
// also checking auth header before start streaming
func handleConnection(remoteConn, localConn net.Conn, cfg Config) {
	defer localConn.Close()
	defer remoteConn.Close()

	var localWriter io.Writer = localConn
	var remoteWriter io.Writer = remoteConn

	var logFileRequest *os.File
	var logFileResponse *os.File

	if cfg.LogTraffic {
		ts := time.Now().UnixNano()
		logFileRequest, _ = os.Create(fmt.Sprintf("log/%d_req.log", ts))
		logFileResponse, _ = os.Create(fmt.Sprintf("log/%d_res.log", ts))
		defer logFileRequest.Close()
		defer logFileResponse.Close()
		localWriter = io.MultiWriter(localConn, logFileRequest)
		remoteWriter = io.MultiWriter(remoteConn, logFileResponse)
	}

	remoteReader := bufio.NewReader(remoteConn)
	var headerBuff bytes.Buffer

	for {
		b, err := remoteReader.ReadByte()
		if err != nil {
			log.Printf("Error while reading: %v. Red bytes: %d", err, headerBuff.Len())
			break
		}
		headerBuff.WriteByte(b)

		if headerBuff.Len() > int(cfg.MaxHttpHeaderSize) {
			break
		}

		// check the end of the header
		if bytes.HasSuffix(headerBuff.Bytes(), []byte("\r\n\r\n")) {
			break
		}
	}

	log.Println()
	log.Println("            HEADERS            ")
	log.Println()
	for i, line := range strings.Split(strings.TrimSpace(headerBuff.String()), "\n") {
		if i == 0 || strings.Contains(line, "Auth") {
			log.Println(line)
		}
	}
	log.Println()

	if !httpAuthenticator.Check(headerBuff.String(), cfg.AuthUser, cfg.AuthPass) {
		log.Println("Unauthorized request. Sleep for 1 second..")
		time.Sleep(time.Second)
		remoteWriter.Write([]byte(httpAuthenticator.UnauthorizedHeader() + "\r\n\r\n"))
		if cfg.LogTraffic {
			logFileRequest.WriteString(headerBuff.String()) // save request headers to the log file
		}
		log.Println("Connection rejected")
		return
	}
	log.Println("OK. Authorized request. Sending response...")

	// Send already red header to the local connection
	io.Copy(localWriter, &headerBuff)

	go func() {
		_, _ = io.Copy(localWriter, remoteReader)
	}()

	_, _ = io.Copy(remoteWriter, localConn)
	// Important! Connection is closed (defer *.Close) when local service sends EOF/closed

}

func keepAliveLoop(client *ssh.Client, interval time.Duration) {
	var lastSuccessKA = time.Now()

	go func() {
		for {
			if lastSuccessKA.Before(time.Now().Add(-interval * 2)) {
				log.Println("keep alive stuck! Closing connection..")
				client.Conn.Close()
				client.Close()
				return
			}
			time.Sleep(time.Second)
		}
	}()

	for {
		//log.Println("keep alive")
		_, _, err := client.Conn.SendRequest("ka", true, nil)
		if err != nil {
			return
		}
		lastSuccessKA = time.Now()
		time.Sleep(interval)
	}
}

func readConfig(filename string) Config {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading YAML file: %s\n", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error parsing YAML file: %s\n", err)
	}

	return config
}

func prepareSshConfig(config Config) *ssh.ClientConfig {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("unable to get user's home directory: %v", err)
	}

	var pkpath = config.PKPath
	if pkpath == "" {
		pkpath = userHomeDir + "/.ssh/id_rsa"
	}

	key, err := os.ReadFile(pkpath)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	sshConfig := ssh.ClientConfig{
		User: strings.Split(config.Remote, "@")[0],
		Auth: []ssh.AuthMethod{
			//ssh.Password("your_ssh_password_here"),
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 5,
	}
	return &sshConfig
}

func checkClientCertificate(tlsConn *tls.Conn, allowedCerts []string) (bool, error) {
	state := tlsConn.ConnectionState()
	if !state.HandshakeComplete {
		err := tlsConn.Handshake()
		if err != nil {
			return false, fmt.Errorf("handshake err: " + err.Error())
		}
		state = tlsConn.ConnectionState()
	}

	log.Println()
	log.Println("            CERT LIST            ")
	log.Println()
	found := false
	for _, cert := range state.PeerCertificates {
		log.Println("Cert Subject: ", cert.Subject)
		log.Println("Issuer: ", cert.Issuer)
		log.Printf("SerialNumber: %X", cert.SerialNumber)

		// check certificate serial number
		if lo.Contains(allowedCerts, fmt.Sprintf("%X", cert.SerialNumber)) {
			found = true
		}
	}

	if !found {
		return false, fmt.Errorf("CERTIFICATE WITH PROPER SERIAL NUMBER NOT FOUND")
	}
	return true, nil
}

func incConnectionsCount(delta int32) {
	atomic.AddInt32(&activeConnections, delta)
	cnt := atomic.LoadInt32(&activeConnections)
	if delta > 0 {
		log.Println("CONNECTIONS++", color.New(color.BgCyan).Sprintf(" %d ", cnt))
	} else {
		log.Println("CONNECTIONS--", color.New(color.BgCyan).Sprintf(" %d ", cnt))
	}
}

func trafficLogCleanupLoop() {
	logsDir := "log"
	for {
		time.Sleep(1 * time.Second)
		files, err := os.ReadDir(logsDir)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".log") {
				continue
			}
			filePath := logsDir + "/" + file.Name()
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				log.Println(err)
				continue
			}
			// delete file older then X
			if time.Now().Sub(fileInfo.ModTime()) > 4*time.Hour {
				err := os.Remove(filePath)
				if err != nil {
					log.Println(err)
				}
			}
		}

		time.Sleep(1 * time.Minute)
	}
}
