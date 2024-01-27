package truncate

import (
	"fmt"
	"os"
)

func TruncateBegin(filename string, newSize int64) error {
	file, err := os.OpenFile(filename, os.O_RDWR, os.ModePerm)

	defer file.Close()
	if err != nil {
		return err
	}
	return TruncateFileBegin(file, newSize)
}

func TruncateFileBegin(file *os.File, newSize int64) error {
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	fileSize := fileInfo.Size()
	if newSize > fileSize {
		return fmt.Errorf("newSize is larger than the file size")
	}

	// Buffer for reading and writing
	var bufSize int64 = 4096
	buffer := make([]byte, bufSize)

	for start := fileSize - newSize; start < fileSize; {
		// calculate block size
		blockSize := myMin(bufSize, fileSize-start)

		// read end of file
		_, err = file.ReadAt(buffer[:blockSize], start)
		if err != nil {
			return err
		}

		// calculate write position
		writePosition := start - (fileSize - newSize)

		// write to the beginning
		_, err = file.WriteAt(buffer[:blockSize], writePosition)
		if err != nil {
			return err
		}

		start += blockSize
	}

	// truncate file
	err = file.Truncate(newSize)
	if err != nil {
		return err
	}

	// move offset to the end of file
	file.Seek(0, 2)

	return nil
}

func myMin(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
