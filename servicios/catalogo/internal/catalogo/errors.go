package catalogo

import "crypto/rand"

type Error struct {
	HTTP     int
	Codigo   string
	Mensaje  string
	Detalles map[string]any
}

func (e *Error) Error() string { return e.Codigo }

func api(http int, codigo, mensaje string, det map[string]any) *Error {
	if det == nil {
		det = map[string]any{}
	}
	return &Error{HTTP: http, Codigo: codigo, Mensaje: mensaje, Detalles: det}
}

func nuevoID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	const hexd = "0123456789abcdef"
	out := make([]byte, 36)
	pos := 0
	for i, c := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[pos] = '-'
			pos++
		}
		out[pos] = hexd[c>>4]
		out[pos+1] = hexd[c&0x0f]
		pos += 2
	}
	return string(out)
}
