package vm

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/daimatz/gojvm/pkg/classfile"
)

// ClassLoader loads .class files by class name.
type ClassLoader interface {
	LoadClass(name string) (*classfile.ClassFile, error)
}

// JmodClassLoader loads classes from a JDK jmod file.
type JmodClassLoader struct {
	JmodPath  string
	Cache     map[string]*classfile.ClassFile
	zipData   []byte
	zipReader *zip.Reader
}

// NewJmodClassLoader creates a new JmodClassLoader.
func NewJmodClassLoader(jmodPath string) *JmodClassLoader {
	return &JmodClassLoader{
		JmodPath: jmodPath,
		Cache:    make(map[string]*classfile.ClassFile),
	}
}

func (cl *JmodClassLoader) ensureZipReader() error {
	if cl.zipReader != nil {
		return nil
	}

	f, err := os.Open(cl.JmodPath)
	if err != nil {
		return fmt.Errorf("jmod: opening %s: %w", cl.JmodPath, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("jmod: stat %s: %w", cl.JmodPath, err)
	}

	data := make([]byte, stat.Size())
	if _, err := io.ReadFull(f, data); err != nil {
		return fmt.Errorf("jmod: reading %s: %w", cl.JmodPath, err)
	}

	cl.zipData = data[4:] // Skip "JM\x01\x00" header
	cl.zipReader, err = zip.NewReader(bytes.NewReader(cl.zipData), int64(len(cl.zipData)))
	if err != nil {
		return fmt.Errorf("jmod: opening zip: %w", err)
	}
	return nil
}

func (cl *JmodClassLoader) LoadClass(name string) (*classfile.ClassFile, error) {
	if cf, ok := cl.Cache[name]; ok {
		return cf, nil
	}

	if err := cl.ensureZipReader(); err != nil {
		return nil, err
	}

	target := "classes/" + name + ".class"
	for _, file := range cl.zipReader.File {
		if file.Name == target {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("jmod: opening %s: %w", target, err)
			}
			defer rc.Close()

			cf, err := classfile.Parse(rc)
			if err != nil {
				return nil, fmt.Errorf("jmod: parsing %s: %w", name, err)
			}
			cl.Cache[name] = cf
			return cf, nil
		}
	}

	return nil, fmt.Errorf("jmod: class %s not found in %s", name, cl.JmodPath)
}

// UserClassLoader loads user classes from the classpath, delegating to the parent first.
type UserClassLoader struct {
	ClassPath string
	Parent    ClassLoader
	Cache     map[string]*classfile.ClassFile
}

// NewUserClassLoader creates a new UserClassLoader.
func NewUserClassLoader(classPath string, parent ClassLoader) *UserClassLoader {
	return &UserClassLoader{
		ClassPath: classPath,
		Parent:    parent,
		Cache:     make(map[string]*classfile.ClassFile),
	}
}

func (cl *UserClassLoader) LoadClass(name string) (*classfile.ClassFile, error) {
	if cf, ok := cl.Cache[name]; ok {
		return cf, nil
	}
	if cf, err := cl.Parent.LoadClass(name); err == nil {
		return cf, nil
	}
	path := filepath.Join(cl.ClassPath, name+".class")
	cf, err := classfile.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("user: class %s not found: %w", name, err)
	}
	cl.Cache[name] = cf
	return cf, nil
}
