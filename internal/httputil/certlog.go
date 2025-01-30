package httputil

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
)

func logPath(path, action string, log logr.Logger) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Error(err, "error in os.Stat()", "path", path)
		return
	}
	if !fi.IsDir() {
		logFile(path, "", fmt.Sprintf("%s file", action), log)
		return
	}
	action = fmt.Sprintf("%s directory", action)
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		log.Error(err, "error in os.ReadDir()", "path", path)
		return
	}
	for _, e := range dirEntries {
		file := filepath.Join(path, e.Name())
		fi, err := os.Stat(file)
		if err != nil {
			log.Error(err, "error in os.Stat()", "file", file)
			continue
		}
		if fi.IsDir() {
			log.Info("ignoring subdirectory", "directory", file)
			continue
		}
		logFile(e.Name(), path, action, log)
	}
}

func logFile(filename, path, action string, log logr.Logger) {
	filepath := filepath.Join(path, filename)
	data, err := os.ReadFile(filepath)
	if err != nil {
		log.Error(err, "error in os.ReadFile()", "file", filename)
		return
	}
	logPem(data, filename, path, action, log)
}

func logPem(data []byte, filename, path, action string, log logr.Logger) {
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			log.Info("error: no block returned from pem.Decode()", "file", filename)
			return
		}
		crt, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Error(err, "error in x509.ParseCertificate()", "file", filename)
			return
		}

		args := []any{}
		if path != "" {
			args = append(args, "directory", path)
		}
		// Find an appopriate certificate identifier
		args = append(args, "file", filename)
		if s := crt.Subject.String(); s != "" {
			args = append(args, "subject", s)
		} else if crt.DNSNames != nil {
			args = append(args, "DNSNames", crt.DNSNames)
		} else if s := crt.SerialNumber.String(); s != "" {
			args = append(args, "serial", s)
		}
		log.Info(action, args...)
	}
}
