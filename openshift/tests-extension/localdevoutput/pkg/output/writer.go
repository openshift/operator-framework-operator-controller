//go:build dev

package output

import (
	"io"
	"os"
	"sync"

	et "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
)

type CleanResultWriter struct {
	lock      sync.Mutex
	formatter *Formatter
	results   []*et.ExtensionTestResult
	writer    io.Writer
}

func NewCleanResultWriter(w io.Writer) *CleanResultWriter {
	if w == nil {
		w = os.Stdout
	}

	formatter := NewFormatter(w)
	formatter.PrintHeader()

	return &CleanResultWriter{
		formatter: formatter,
		results:   make([]*et.ExtensionTestResult, 0),
		writer:    w,
	}
}

func (w *CleanResultWriter) SetTotalTests(total int) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.formatter.SetTotalTests(total)
}

func (w *CleanResultWriter) Write(result *et.ExtensionTestResult) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if result == nil {
		return
	}

	w.results = append(w.results, result)
	w.formatter.PrintTestStart(result.Name)
	w.formatter.PrintTestResult(result)
}

func (w *CleanResultWriter) Flush() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.formatter.PrintSummary()
	return nil
}

func (w *CleanResultWriter) HasFailures() bool {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.formatter.HasFailures()
}
