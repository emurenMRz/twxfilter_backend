package main

type DaemonError struct {
	err  error
	code int
	text string
}

func (e *DaemonError) Code() int {
	return e.code
}

func (e *DaemonError) Error() string {
	if e.err == nil {
		if e.text == "" {
			return "daemon error"
		}
		return e.text
	}
	return e.err.Error()
}

func (e *DaemonError) Text() string {
	if e.err == nil {
		if e.text == "" {
			return "daemon error"
		}
		return e.text
	}
	if e.text == "" {
		return e.text
	}
	return e.err.Error()
}

func NewDaemonError(err error, code int, text string) error {
	return &DaemonError{err: err, code: code, text: text}
}
