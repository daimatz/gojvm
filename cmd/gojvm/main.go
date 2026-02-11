package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daimatz/gojvm/pkg/vm"
)

func findJmodPath() string {
	// 1. Explicit env var
	if env := os.Getenv("JAVA_BASE_JMOD"); env != "" {
		return env
	}
	// 2. JAVA_HOME
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		p := filepath.Join(javaHome, "jmods", "java.base.jmod")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 3. Glob fallback
	matches, _ := filepath.Glob("/usr/lib/jvm/java-*-openjdk-*/jmods/java.base.jmod")
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gojvm <classfile>\n")
		os.Exit(1)
	}

	filename := os.Args[1]
	dir := filepath.Dir(filename)
	className := strings.TrimSuffix(filepath.Base(filename), ".class")

	jmodPath := findJmodPath()
	if jmodPath == "" {
		fmt.Fprintf(os.Stderr, "Error: could not find java.base.jmod. Set JAVA_HOME or JAVA_BASE_JMOD.\n")
		os.Exit(1)
	}

	bootstrap := vm.NewJmodClassLoader(jmodPath)
	userCL := vm.NewUserClassLoader(dir, bootstrap)

	v := vm.NewVM(userCL)

	if err := v.Execute(className); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing: %v\n", err)
		os.Exit(1)
	}
}
