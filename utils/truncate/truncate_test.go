package truncate

import (
	"fmt"
	"log"
	"os"
	"testing"
)

var testFileName = "test_file.txt"

func TestMain(m *testing.M) {

	file, err := os.Create(testFileName)
	if err != nil {
		log.Printf("Cannot create temp file: %v", err)
		os.Exit(1)
	}

	for i := 0; i < 1000*1000; i++ {
		_, err = file.WriteString(fmt.Sprintf("Hello world! %0.10d \n", i))
		if err != nil {
			log.Panicf("Cannot write to temp file: %v", err)
		}
	}

	file.Close()

	// run all tests
	exitVal := m.Run()

	os.Remove(testFileName)

	os.Exit(exitVal)
}

func TestTruncateBegin(t *testing.T) {
	var err error

	err = TruncateBegin(testFileName, 10*1024*1024)
	if err != nil {
		t.Fatalf("Cannot truncate: %v", err)
	}
}

func TestTruncateFileBegin(t *testing.T) {

	content := []byte("Hello, this is a test file for TruncateFileBegin ( ) function.")

	// define test cases
	tests := []struct {
		n        int
		wantErr  bool
		expected string
	}{
		{1, false, "."},
		{5, false, "tion."},
		{6, false, "ction."},
		{31, false, "TruncateFileBegin ( ) function."},
		{40, false, "file for TruncateFileBegin ( ) function."},
		{len(content), false, string(content)},
		{len(content) + 1, true, ""},
		{0, false, ""}, // empty file
	}

	for index, tt := range tests {

		tempFile, err := os.CreateTemp("", "testfile_*.txt")
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		if _, err := tempFile.Write(content); err != nil {
			t.Fatalf("failed to write to the file : %v", err)
		}

		err = TruncateFileBegin(tempFile, int64(tt.n))

		if (err != nil) != tt.wantErr {
			t.Errorf("test #%d: error = %v, wantErr %v, expected '%s'", index, err, tt.wantErr, tt.expected)
			continue
		}

		if !tt.wantErr {
			result, _ := os.ReadFile(tempFile.Name())
			if string(result) != tt.expected {
				t.Errorf("test #%d: result = '%s', want '%s'", index, string(result), tt.expected)
			}
		}

		pos, _ := tempFile.Seek(0, 1)
		info, _ := tempFile.Stat()
		if pos != info.Size() {
			t.Errorf("test #%d: wrong pos'%d' expected '%d'", index, pos, info.Size())
		}

		tempFile.Close()
		os.Remove(tempFile.Name())
	}
}
