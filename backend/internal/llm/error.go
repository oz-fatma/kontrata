package llm

import "errors"

// CallError bir LLM çağrısının sınıflandırılmış sonucudur.
type CallError struct {
	Tipi string
	Cold bool
	err  error
}

func (e *CallError) Error() string {
	if e == nil || e.err == nil {
		return ErrUnavailable.Error()
	}
	return e.err.Error()
}

func (e *CallError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func wrapCallError(kind string, err error) error {
	if err == nil {
		return nil
	}
	tipi, cold := hataTipiFromKind(kind)
	return &CallError{Tipi: tipi, Cold: cold, err: err}
}

func hataTipiFromKind(kind string) (string, bool) {
	switch kind {
	case "ok", "":
		return HataYok, false
	case "503":
		return HataHTTP5xx, true
	case "zaman_asimi", "iptal":
		return HataTimeout, false
	case "yanit_ayristirma":
		return HataParse, false
	}
	if len(kind) >= 6 && kind[:5] == "http_" {
		code := kind[5:]
		if len(code) > 0 && code[0] == '4' {
			return HataHTTP4xx, false
		}
		return HataHTTP5xx, false
	}
	return HataHTTP5xx, false
}

func classifyError(err error) (tipi string, cold bool) {
	if err == nil {
		return HataYok, false
	}
	var ce *CallError
	if errors.As(err, &ce) && ce != nil {
		if ce.Tipi == "" {
			return HataHTTP5xx, ce.Cold
		}
		return ce.Tipi, ce.Cold
	}
	if errors.Is(err, ErrColdStart) {
		return HataHTTP5xx, true
	}
	if errors.Is(err, ErrUnavailable) {
		return HataHTTP5xx, false
	}
	return HataHTTP5xx, false
}
