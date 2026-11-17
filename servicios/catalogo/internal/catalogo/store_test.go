package catalogo

import (
	"context"
	"os"
	"testing"
)

func TestReservaIdempotenteYCompensacion(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no está definida")
	}
	ctx := context.Background()
	store, err := Conectar(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrar(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `TRUNCATE reservas, productos, productores CASCADE`); err != nil {
		t.Fatal(err)
	}
	productor, err := store.CrearProductor(ctx, NuevoProductor{
		Nombre: "Huerta de prueba", Distrito: "Surco", Telefono: "+51940000001", Descripcion: "Prueba",
	})
	if err != nil {
		t.Fatal(err)
	}
	stock, errCant := ParseCantidad("2.000", "kg", "stock", true)
	if errCant != nil {
		t.Fatal(errCant)
	}
	producto, err := store.CrearProducto(ctx, NuevoProducto{
		ProductorID: productor.ID, Nombre: "Palta", Categoria: "frutas", Unidad: "kg",
		PrecioCentavos: 890, Stock: stock, Descripcion: "Prueba",
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := "ped:prueba:prod:palta01"
	primera, status, err := store.Reservar(ctx, producto.ID, "1.500", ref)
	if err != nil || status != 201 {
		t.Fatalf("alta status=%d err=%v", status, err)
	}
	if primera.PrecioCentavos != 890 || primera.Cantidad != "1.500" {
		t.Fatalf("snapshot inesperado: %+v", primera)
	}
	segunda, status, err := store.Reservar(ctx, producto.ID, "1.5", ref)
	if err != nil || status != 200 || segunda.ID != primera.ID {
		t.Fatalf("reintento status=%d id=%s err=%v", status, segunda.ID, err)
	}
	actual, err := store.ObtenerProducto(ctx, producto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Stock != "0.500" {
		t.Fatalf("stock = %s", actual.Stock)
	}
	if _, _, err := store.Reservar(ctx, producto.ID, "1.000", "ped:prueba:prod:palta02"); err == nil || err.(*Error).Codigo != "STOCK_INSUFICIENTE" {
		t.Fatalf("se esperaba STOCK_INSUFICIENTE, err=%v", err)
	}
	comp, err := store.Compensar(ctx, primera.ID)
	if err != nil || comp.Estado != "compensada" {
		t.Fatal(err, comp.Estado)
	}
	otra, err := store.Compensar(ctx, primera.ID)
	if err != nil || otra.Estado != "compensada" {
		t.Fatal("compensar debía ser idempotente", err)
	}
	reactivada, status, err := store.Reservar(ctx, producto.ID, "1.500", ref)
	if err != nil || status != 200 || reactivada.Estado != "activa" || reactivada.ID != primera.ID {
		t.Fatalf("reactivación status=%d %+v err=%v", status, reactivada, err)
	}
	if _, _, err := store.Reservar(ctx, producto.ID, "0.250", ref); err == nil || err.(*Error).Codigo != "REFERENCIA_CONFLICTO" {
		t.Fatalf("se esperaba REFERENCIA_CONFLICTO, err=%v", err)
	}
}
