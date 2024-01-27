
# go-tunnel

Expose your local HTTP/WebSocket ports to a public server securely.

Features:
- Connection is protected with SSL.
- All the SSL configurations are handled locally.
- There is no need to configure Nginx/Let's Encrypt, etc., on your public server.
- Protection with a client's certificate in the browser.
- Extra protection with Digest/Basic authorization.



 
### How it works?
It creates SSH tunnels from your local machine to your public server with port forwarding (similar to `ssh -R {ip:port}{ip:port}`)
and adds SSL encryption and Basic/Digest authorization over it.
