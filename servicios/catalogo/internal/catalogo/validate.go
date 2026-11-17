package catalogo

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

var (
	categorias = map[string]struct{}{
		"frutas": {}, "verduras": {}, "lacteos": {}, "granos": {}, "miel": {}, "otros": {},
	}
	unidades = map[string]struct{}{
		"kg": {}, "unidad": {}, "litro": {}, "atado": {},
	}
	unidadesEnteras = map[string]struct{}{
		"unidad": {}, "atado": {},
	}
	telefonoRe   = regexp.MustCompile(`^\+51[0-9]{9}$`)
	cantidadRe   = regexp.MustCompile(`^(?:0|[1-9][0-9]{0,6})(?:\.[0-9]{1,3})?$`)
	referenciaRe = regexp.MustCompile(`^[A-Za-z0-9:._-]{8,120}$`)
	uuidRe       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func NormalizarID(s string) (string, *Error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !uuidRe.MatchString(s) {
		return "", solicitud("El identificador no es un UUID.", map[string]any{"campo": "id"})
	}
	return s, nil
}

func ValidarReferencia(s string) *Error {
	if !referenciaRe.MatchString(s) {
		return solicitud("La referencia debe tener entre 8 y 120 caracteres y solo letras, números, y los signos : . _ -.", map[string]any{"campo": "referencia"})
	}
	return nil
}

func ValidarCategoria(s string) *Error {
	if _, ok := categorias[s]; !ok {
		return solicitud("La categoría no es válida.", map[string]any{"campo": "categoria"})
	}
	return nil
}

func ValidarUnidad(s string) *Error {
	if _, ok := unidades[s]; !ok {
		return solicitud("La unidad no es válida.", map[string]any{"campo": "unidad"})
	}
	return nil
}

func ValidarTelefono(s string) (string, *Error) {
	s = strings.TrimSpace(s)
	if !telefonoRe.MatchString(s) {
		return "", solicitud("El teléfono debe empezar con +51 y tener 9 dígitos.", map[string]any{"campo": "telefono"})
	}
	return s, nil
}

func ValidarPrecio(v int) *Error {
	if v <= 0 || v > 100000000 {
		return solicitud("El precio debe estar entre 1 y 100000000 céntimos.", map[string]any{"campo": "precio_centavos"})
	}
	return nil
}

func Texto(s, campo string, min, max int) (string, *Error) {
	s = strings.TrimSpace(s)
	for _, r := range s {
		if r < 32 {
			return "", solicitud("El texto tiene caracteres de control.", map[string]any{"campo": campo})
		}
	}
	n := utf8.RuneCountInString(s)
	if n < min || n > max {
		return "", solicitud("La longitud del campo no es válida.", map[string]any{"campo": campo, "min": min, "max": max})
	}
	return s, nil
}

// ParseCantidad acepta un decimal en texto. permitirCero distingue stock (true, tope 1000000) de una reserva (false, tope 10000).
func ParseCantidad(raw, unidad, campo string, permitirCero bool) (decimal.Decimal, *Error) {
	raw = strings.TrimSpace(raw)
	if !cantidadRe.MatchString(raw) {
		return decimal.Decimal{}, solicitud("El número debe ser positivo, con hasta 3 decimales y sin notación científica.", map[string]any{"campo": campo})
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Decimal{}, solicitud("El número no se pudo leer.", map[string]any{"campo": campo})
	}
	if d.Sign() < 0 || (!permitirCero && !d.IsPositive()) {
		return decimal.Decimal{}, solicitud("La cantidad debe ser mayor que cero.", map[string]any{"campo": campo})
	}
	max := decimal.NewFromInt(10000)
	if permitirCero {
		max = decimal.NewFromInt(1000000)
	}
	if d.Cmp(max) > 0 {
		return decimal.Decimal{}, solicitud("El número supera el máximo permitido.", map[string]any{"campo": campo, "max": max.String()})
	}
	if _, entera := unidadesEnteras[unidad]; entera && !d.Equal(d.Truncate(0)) {
		return decimal.Decimal{}, solicitud("Esta unidad solo admite cantidades enteras.", map[string]any{"campo": campo, "unidad": unidad})
	}
	return d, nil
}

func SubtotalCentavos(precio int, cantidad decimal.Decimal) int {
	v := decimal.NewFromInt(int64(precio)).Mul(cantidad).Round(0)
	return int(v.IntPart())
}

func FormatoCantidad(d decimal.Decimal) string {
	return d.StringFixed(3)
}

func solicitud(mensaje string, det map[string]any) *Error {
	return &Error{HTTP: 400, Codigo: "SOLICITUD_INVALIDA", Mensaje: mensaje, Detalles: det}
}
