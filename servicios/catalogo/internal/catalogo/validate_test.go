package catalogo

import "testing"

func TestSubtotalRedondeo(t *testing.T) {
	palta, err := ParseCantidad("1.5", "kg", "cantidad", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := SubtotalCentavos(890, palta); got != 1335 {
		t.Fatalf("890 * 1.5 = %d, se esperaba 1335", got)
	}
	tercio, err := ParseCantidad("0.335", "kg", "cantidad", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := SubtotalCentavos(100, tercio); got != 34 {
		t.Fatalf("100 * 0.335 = %d, se esperaba 34", got)
	}
	medio, err := ParseCantidad("0.5", "kg", "cantidad", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := SubtotalCentavos(1, medio); got != 1 {
		t.Fatalf("1 * 0.5 = %d, se esperaba 1", got)
	}
}

func TestCantidadEntera(t *testing.T) {
	if _, err := ParseCantidad("2.000", "unidad", "cantidad", false); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseCantidad("2.500", "atado", "cantidad", false); err == nil {
		t.Fatal("un atado fraccionario debía rechazarse")
	}
	if _, err := ParseCantidad("0", "kg", "cantidad", false); err == nil {
		t.Fatal("la reserva no admite cero")
	}
	if _, err := ParseCantidad("0.000", "kg", "stock", true); err != nil {
		t.Fatal(err)
	}
}

func TestTelefono(t *testing.T) {
	if _, err := ValidarTelefono("+51941111001"); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidarTelefono("941111001"); err == nil {
		t.Fatal("sin prefijo debía rechazarse")
	}
}
