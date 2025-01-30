package httputil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
)

const (
	defaultLogLevel = 0 // 4
)

func DumpCertificates(path string, log logr.Logger) {
	var paths []string
	if path == "" {
		paths = []string{"/etc/docker/certs.d", "/etc/containers/certs.d"}
	} else {
		paths = []string{path}
	}
	for _, path := range paths {
		fi, err := os.Stat(path)
		if err != nil {
			log.Error(err, "statting directory", "directory", path)
			continue
		}
		if !fi.IsDir() {
			log.V(defaultLogLevel).V(1).Info("not a directory", "directory", path)
			continue
		}
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			log.Error(err, "reading directory", "directory", path)
			continue
		}
		count := 0
		for _, e := range dirEntries {
			hostPath := filepath.Join(path, e.Name())
			fi, err := os.Stat(hostPath)
			count++
			if err != nil {
				log.Error(err, "dumping certs", "path", hostPath)
				continue
			}
			if !fi.IsDir() {
				log.V(defaultLogLevel).V(1).Info("ignoring non-directory", "path", hostPath)
				continue
			}
			logPath(hostPath, "dump docker certs", log)
		}
	}
}

func LogUnverifiedCertificate(err error, log logr.Logger) bool {
	for err != nil {
		var cvErr *tls.CertificateVerificationError
		if errors.As(err, &cvErr) {
			n := 1
			for _, cert := range cvErr.UnverifiedCertificates {
				log.Error(err, "unverified cert", "n", n, "subject", cert.Subject, "issuer", cert.Issuer, "DNSNames", cert.DNSNames, "serial", cert.SerialNumber)
				n = n + 1
			}
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

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
			log.V(defaultLogLevel).V(1).Info("ignoring subdirectory", "directory", file)
			continue
		}
		logFile(e.Name(), path, action, log)
	}
}

func logFile(filename, path, action string, log logr.Logger) {
	filepath := filepath.Join(path, filename)
	_, err := os.Stat(filepath)
	if err != nil {
		log.Error(err, "statting file", "file", filepath)
		return
	}
	data, err := os.ReadFile(filepath)
	if err != nil {
		log.Error(err, "error in os.ReadFile()", "file", filepath)
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
		log.V(defaultLogLevel).Info(action, args...)
	}
}
