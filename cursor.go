package main

import (
	"fmt"
	"os"
)

// Cursor stores and retrieve the cursor, a string that uniquely describes the position of an entry in the journal.
type Cursor interface {
	Get() (string, error)
	Set(string) error
}

// FilebasedCursor stores the cursor in a file.
type FilebasedCursor struct {
	file *os.File
}

func NewFilebasedCursor(fileName string) (*FilebasedCursor, error) {
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	return &FilebasedCursor{
		file: f,
	}, nil
}

func (c *FilebasedCursor) Get() (string, error) {
	var cursor string
	_, err := c.file.Seek(0, 0)
	if err != nil {
		return "", err
	}
	if _, err := fmt.Fscanf(c.file, "%s", &cursor); err != nil {
		return "", err
	}
	return cursor, nil
}

func (c *FilebasedCursor) Set(v string) error {
	if _, err := c.file.Seek(0, 0); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.file, "%s", v); err != nil {
		return err
	}
	return nil
}

func (c *FilebasedCursor) Close() error {
	return c.file.Close()
}
