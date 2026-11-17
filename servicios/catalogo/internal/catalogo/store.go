package catalogo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Store struct {
	pool *pgxpool.Pool
}

func Conectar(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 5
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) Migrar(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schemaSQL); err != nil {
		return err
	}
	var version int
	err := s.pool.QueryRow(ctx, `SELECT version FROM esquema LIMIT 1`).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = s.pool.Exec(ctx, `INSERT INTO esquema (version) VALUES (1)`)
		return err
	}
	if err != nil {
		return err
	}
	if version != 1 {
		return fmt.Errorf("esquema de catálogo en versión %d; esta imagen solo acepta la versión 1", version)
	}
	return nil
}

type Productor struct {
	ID               string    `json:"id"`
	Nombre           string    `json:"nombre"`
	Distrito         string    `json:"distrito"`
	Telefono         string    `json:"telefono"`
	Descripcion      string    `json:"descripcion"`
	ProductosActivos int       `json:"productos_activos"`
	CreadoEn         time.Time `json:"creado_en"`
}

type ProductorResumen struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Distrito string `json:"distrito"`
}

type Producto struct {
	ID             string           `json:"id"`
	Productor      ProductorResumen `json:"productor"`
	Nombre         string           `json:"nombre"`
	Categoria      string           `json:"categoria"`
	Unidad         string           `json:"unidad"`
	PrecioCentavos int              `json:"precio_centavos"`
	Stock          string           `json:"stock"`
	Activo         bool             `json:"activo"`
	Descripcion    string           `json:"descripcion"`
	CreadoEn       time.Time        `json:"creado_en"`
	ActualizadoEn  time.Time        `json:"actualizado_en"`
}

type Reserva struct {
	ID             string    `json:"id"`
	ProductoID     string    `json:"producto_id"`
	ProductorID    string    `json:"productor_id"`
	Nombre         string    `json:"nombre"`
	Unidad         string    `json:"unidad"`
	PrecioCentavos int       `json:"precio_centavos"`
	Cantidad       string    `json:"cantidad"`
	Referencia     string    `json:"referencia"`
	Estado         string    `json:"estado"`
	CreadoEn       time.Time `json:"creado_en"`
}

type Pagina[T any] struct {
	Datos      []T `json:"datos"`
	Paginacion struct {
		Limite         int `json:"limite"`
		Desplazamiento int `json:"desplazamiento"`
		Total          int `json:"total"`
	} `json:"paginacion"`
}

type FiltroProductos struct {
	Q              string
	Categoria      string
	ProductorID    string
	Activo         *bool
	Limite         int
	Desplazamiento int
}

func (s *Store) ListarProductores(ctx context.Context, limite, desplazamiento int) (Pagina[Productor], error) {
	var pagina Pagina[Productor]
	pagina.Paginacion.Limite = limite
	pagina.Paginacion.Desplazamiento = desplazamiento
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM productores`).Scan(&pagina.Paginacion.Total); err != nil {
		return pagina, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT pr.id::text, pr.nombre, pr.distrito, pr.telefono, pr.descripcion, pr.creado_en,
		       (SELECT count(*) FROM productos p WHERE p.productor_id = pr.id AND p.activo)
		FROM productores pr
		ORDER BY pr.nombre ASC, pr.id ASC
		LIMIT $1 OFFSET $2`, limite, desplazamiento)
	if err != nil {
		return pagina, err
	}
	defer rows.Close()
	pagina.Datos = []Productor{}
	for rows.Next() {
		var p Productor
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Distrito, &p.Telefono, &p.Descripcion, &p.CreadoEn, &p.ProductosActivos); err != nil {
			return pagina, err
		}
		pagina.Datos = append(pagina.Datos, p)
	}
	return pagina, rows.Err()
}

type NuevoProductor struct {
	Nombre, Distrito, Telefono, Descripcion string
}

func (s *Store) CrearProductor(ctx context.Context, in NuevoProductor) (Productor, error) {
	id := nuevoID()
	var p Productor
	err := s.pool.QueryRow(ctx, `
		INSERT INTO productores (id, nombre, distrito, telefono, descripcion)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, nombre, distrito, telefono, descripcion, creado_en`,
		id, in.Nombre, in.Distrito, in.Telefono, in.Descripcion,
	).Scan(&p.ID, &p.Nombre, &p.Distrito, &p.Telefono, &p.Descripcion, &p.CreadoEn)
	return p, err
}

func (s *Store) ObtenerProductor(ctx context.Context, id string) (Productor, error) {
	var p Productor
	err := s.pool.QueryRow(ctx, `
		SELECT pr.id::text, pr.nombre, pr.distrito, pr.telefono, pr.descripcion, pr.creado_en,
		       (SELECT count(*) FROM productos p WHERE p.productor_id = pr.id AND p.activo)
		FROM productores pr WHERE pr.id = $1`, id,
	).Scan(&p.ID, &p.Nombre, &p.Distrito, &p.Telefono, &p.Descripcion, &p.CreadoEn, &p.ProductosActivos)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, api(404, "NO_ENCONTRADO", "No existe el productor.", map[string]any{"recurso": "productor", "id": id})
	}
	return p, err
}

type ParcheProductor struct {
	Nombre, Distrito, Telefono, Descripcion *string
}

func (s *Store) ActualizarProductor(ctx context.Context, id string, in ParcheProductor) (Productor, error) {
	sets := make([]string, 0, 4)
	args := []any{id}
	add := func(col string, v *string) {
		if v == nil {
			return
		}
		args = append(args, *v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	add("nombre", in.Nombre)
	add("distrito", in.Distrito)
	add("telefono", in.Telefono)
	add("descripcion", in.Descripcion)
	if len(sets) == 0 {
		return Productor{}, solicitud("No hay campos para actualizar.", nil)
	}
	q := `UPDATE productores SET ` + strings.Join(sets, ", ") + ` WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, args...)
	if err != nil {
		return Productor{}, err
	}
	if tag.RowsAffected() == 0 {
		return Productor{}, api(404, "NO_ENCONTRADO", "No existe el productor.", map[string]any{"recurso": "productor", "id": id})
	}
	return s.ObtenerProductor(ctx, id)
}

func (s *Store) EliminarProductor(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM productores WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return api(404, "NO_ENCONTRADO", "No existe el productor.", map[string]any{"recurso": "productor", "id": id})
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM productos WHERE productor_id = $1`, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return api(409, "PRODUCTOR_CON_PRODUCTOS", "El productor todavía tiene productos. Desactívalos antes de borrarlo.", map[string]any{"productos": n})
	}
	if _, err := tx.Exec(ctx, `DELETE FROM productores WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListarProductos(ctx context.Context, f FiltroProductos) (Pagina[Producto], error) {
	var pagina Pagina[Producto]
	pagina.Datos = []Producto{}
	pagina.Paginacion.Limite = f.Limite
	pagina.Paginacion.Desplazamiento = f.Desplazamiento
	where := []string{"TRUE"}
	args := []any{}
	if f.Q != "" {
		args = append(args, "%"+escaparLike(f.Q)+"%")
		where = append(where, fmt.Sprintf("p.nombre ILIKE $%d ESCAPE '\\'", len(args)))
	}
	if f.Categoria != "" {
		args = append(args, f.Categoria)
		where = append(where, fmt.Sprintf("p.categoria = $%d", len(args)))
	}
	if f.ProductorID != "" {
		args = append(args, f.ProductorID)
		where = append(where, fmt.Sprintf("p.productor_id = $%d", len(args)))
	}
	if f.Activo != nil {
		args = append(args, *f.Activo)
		where = append(where, fmt.Sprintf("p.activo = $%d", len(args)))
	}
	filtro := strings.Join(where, " AND ")
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM productos p WHERE `+filtro, args...).Scan(&pagina.Paginacion.Total); err != nil {
		return pagina, err
	}
	args = append(args, f.Limite, f.Desplazamiento)
	rows, err := s.pool.Query(ctx, `
		SELECT p.id::text, pr.id::text, pr.nombre, pr.distrito, p.nombre, p.categoria, p.unidad,
		       p.precio_centavos, p.stock::text, p.activo, p.descripcion, p.creado_en, p.actualizado_en
		FROM productos p
		JOIN productores pr ON pr.id = p.productor_id
		WHERE `+filtro+`
		ORDER BY p.nombre ASC, p.id ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return pagina, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanProducto(rows)
		if err != nil {
			return pagina, err
		}
		pagina.Datos = append(pagina.Datos, p)
	}
	return pagina, rows.Err()
}

type NuevoProducto struct {
	ProductorID, Nombre, Categoria, Unidad, Descripcion string
	PrecioCentavos                                      int
	Stock                                               decimal.Decimal
}

func (s *Store) CrearProducto(ctx context.Context, in NuevoProducto) (Producto, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Producto{}, err
	}
	defer tx.Rollback(ctx)
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM productores WHERE id = $1)`, in.ProductorID).Scan(&ok); err != nil {
		return Producto{}, err
	}
	if !ok {
		return Producto{}, api(404, "NO_ENCONTRADO", "No existe el productor.", map[string]any{"recurso": "productor", "id": in.ProductorID})
	}
	id := nuevoID()
	if _, err := tx.Exec(ctx, `
		INSERT INTO productos (id, productor_id, nombre, categoria, unidad, precio_centavos, stock, descripcion)
		VALUES ($1, $2, $3, $4, $5, $6, $7::numeric, $8)`,
		id, in.ProductorID, in.Nombre, in.Categoria, in.Unidad, in.PrecioCentavos, FormatoCantidad(in.Stock), in.Descripcion,
	); err != nil {
		return Producto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Producto{}, err
	}
	return s.ObtenerProducto(ctx, id)
}

func (s *Store) ObtenerProducto(ctx context.Context, id string) (Producto, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id::text, pr.id::text, pr.nombre, pr.distrito, p.nombre, p.categoria, p.unidad,
		       p.precio_centavos, p.stock::text, p.activo, p.descripcion, p.creado_en, p.actualizado_en
		FROM productos p
		JOIN productores pr ON pr.id = p.productor_id
		WHERE p.id = $1`, id)
	if err != nil {
		return Producto{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return Producto{}, err
		}
		return Producto{}, api(404, "NO_ENCONTRADO", "No existe el producto.", map[string]any{"recurso": "producto", "id": id})
	}
	return scanProducto(rows)
}

type ParcheProducto struct {
	Nombre, Categoria, Unidad, Descripcion *string
	PrecioCentavos                         *int
	Stock                                  *decimal.Decimal
	Activo                                 *bool
}

func (s *Store) ActualizarProducto(ctx context.Context, id string, in ParcheProducto) (Producto, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Producto{}, err
	}
	defer tx.Rollback(ctx)
	var unidad string
	var stockTxt string
	err = tx.QueryRow(ctx, `SELECT unidad, stock::text FROM productos WHERE id = $1 FOR UPDATE`, id).Scan(&unidad, &stockTxt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Producto{}, api(404, "NO_ENCONTRADO", "No existe el producto.", map[string]any{"recurso": "producto", "id": id})
	}
	if err != nil {
		return Producto{}, err
	}
	if in.Unidad != nil {
		unidad = *in.Unidad
	}
	stock, errDec := decimal.NewFromString(stockTxt)
	if errDec != nil {
		return Producto{}, errDec
	}
	if in.Stock != nil {
		stock = *in.Stock
	}
	if _, err := ParseCantidad(FormatoCantidad(stock), unidad, "stock", true); err != nil {
		return Producto{}, err
	}
	sets := []string{"actualizado_en = now()"}
	args := []any{id}
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if in.Nombre != nil {
		add("nombre", *in.Nombre)
	}
	if in.Categoria != nil {
		add("categoria", *in.Categoria)
	}
	if in.Unidad != nil {
		add("unidad", *in.Unidad)
	}
	if in.Descripcion != nil {
		add("descripcion", *in.Descripcion)
	}
	if in.PrecioCentavos != nil {
		add("precio_centavos", *in.PrecioCentavos)
	}
	if in.Stock != nil {
		add("stock", FormatoCantidad(*in.Stock))
		sets[len(sets)-1] = fmt.Sprintf("stock = $%d::numeric", len(args))
	}
	if in.Activo != nil {
		add("activo", *in.Activo)
	}
	if len(sets) == 1 {
		return Producto{}, solicitud("No hay campos para actualizar.", nil)
	}
	if _, err := tx.Exec(ctx, `UPDATE productos SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...); err != nil {
		return Producto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Producto{}, err
	}
	return s.ObtenerProducto(ctx, id)
}

func (s *Store) DesactivarProducto(ctx context.Context, id string) (Producto, error) {
	falso := false
	return s.ActualizarProducto(ctx, id, ParcheProducto{Activo: &falso})
}

func (s *Store) Reservar(ctx context.Context, productoID, cantidadRaw, referencia string) (Reserva, int, error) {
	if _, errV := parseCantidadLibre(cantidadRaw); errV != nil {
		return Reserva{}, 0, errV
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reserva{}, 0, err
	}
	defer tx.Rollback(ctx)

	var activo bool
	var unidad, nombre, productorID, stockTxt string
	var precio int
	err = tx.QueryRow(ctx, `
		SELECT nombre, productor_id::text, unidad, precio_centavos, stock::text, activo
		FROM productos WHERE id = $1 FOR UPDATE`, productoID,
	).Scan(&nombre, &productorID, &unidad, &precio, &stockTxt, &activo)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reserva{}, 0, api(404, "NO_ENCONTRADO", "No existe el producto.", map[string]any{"recurso": "producto", "id": productoID})
	}
	if err != nil {
		return Reserva{}, 0, err
	}
	cant, errV := ParseCantidad(cantidadRaw, unidad, "cantidad", false)
	if errV != nil {
		return Reserva{}, 0, errV
	}
	stock, err := decimal.NewFromString(stockTxt)
	if err != nil {
		return Reserva{}, 0, err
	}

	var reservaID, estado, snapNombre, snapProductor, snapUnidad, snapCant string
	var snapPrecio int
	var creado time.Time
	err = tx.QueryRow(ctx, `
		SELECT id::text, estado, nombre, productor_id::text, unidad, precio_centavos, cantidad::text, creado_en
		FROM reservas WHERE referencia = $1 FOR UPDATE`, referencia,
	).Scan(&reservaID, &estado, &snapNombre, &snapProductor, &snapUnidad, &snapPrecio, &snapCant, &creado)
	encontrada := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Reserva{}, 0, err
	}
	if encontrada {
		prev, err := decimal.NewFromString(snapCant)
		if err != nil {
			return Reserva{}, 0, err
		}
		var dueño string
		if err := tx.QueryRow(ctx, `SELECT producto_id::text FROM reservas WHERE id = $1`, reservaID).Scan(&dueño); err != nil {
			return Reserva{}, 0, err
		}
		if dueño != productoID || !prev.Equal(cant) {
			return Reserva{}, 0, api(409, "REFERENCIA_CONFLICTO", "La referencia ya se usó con otro producto o cantidad.", map[string]any{"referencia": referencia})
		}
		if estado == "activa" {
			if err := tx.Commit(ctx); err != nil {
				return Reserva{}, 0, err
			}
			return Reserva{
				ID: reservaID, ProductoID: productoID, ProductorID: snapProductor, Nombre: snapNombre,
				Unidad: snapUnidad, PrecioCentavos: snapPrecio, Cantidad: FormatoCantidad(prev),
				Referencia: referencia, Estado: "activa", CreadoEn: creado.UTC(),
			}, 200, nil
		}
	}

	if !activo {
		return Reserva{}, 0, api(409, "PRODUCTO_INACTIVO", "El producto no está a la venta.", map[string]any{"producto_id": productoID})
	}
	if stock.Cmp(cant) < 0 {
		return Reserva{}, 0, api(409, "STOCK_INSUFICIENTE", "No hay stock suficiente.", map[string]any{
			"producto_id": productoID, "stock": FormatoCantidad(stock), "solicitado": FormatoCantidad(cant),
		})
	}
	if _, err := tx.Exec(ctx, `UPDATE productos SET stock = stock - $2::numeric, actualizado_en = now() WHERE id = $1`, productoID, FormatoCantidad(cant)); err != nil {
		return Reserva{}, 0, err
	}
	status := 201
	if encontrada {
		status = 200
		if _, err := tx.Exec(ctx, `
			UPDATE reservas
			SET estado = 'activa', compensado_en = NULL, nombre = $2, productor_id = $3, unidad = $4, precio_centavos = $5, creado_en = now()
			WHERE id = $1`, reservaID, nombre, productorID, unidad, precio); err != nil {
			return Reserva{}, 0, err
		}
	} else {
		reservaID = nuevoID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO reservas (id, producto_id, productor_id, nombre, unidad, precio_centavos, cantidad, referencia, estado)
			VALUES ($1, $2, $3, $4, $5, $6, $7::numeric, $8, 'activa')`,
			reservaID, productoID, productorID, nombre, unidad, precio, FormatoCantidad(cant), referencia,
		); err != nil {
			return Reserva{}, 0, err
		}
	}
	var out Reserva
	if err := tx.QueryRow(ctx, `
		SELECT id::text, producto_id::text, productor_id::text, nombre, unidad, precio_centavos, cantidad::text, referencia, estado, creado_en
		FROM reservas WHERE id = $1`, reservaID,
	).Scan(&out.ID, &out.ProductoID, &out.ProductorID, &out.Nombre, &out.Unidad, &out.PrecioCentavos, &out.Cantidad, &out.Referencia, &out.Estado, &out.CreadoEn); err != nil {
		return Reserva{}, 0, err
	}
	out.Cantidad = mustFixed(out.Cantidad)
	out.CreadoEn = out.CreadoEn.UTC()
	if err := tx.Commit(ctx); err != nil {
		return Reserva{}, 0, err
	}
	return out, status, nil
}

func (s *Store) Compensar(ctx context.Context, reservaID string) (Reserva, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reserva{}, err
	}
	defer tx.Rollback(ctx)
	var productoID string
	err = tx.QueryRow(ctx, `SELECT producto_id::text FROM reservas WHERE id = $1`, reservaID).Scan(&productoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reserva{}, api(404, "NO_ENCONTRADO", "No existe la reserva.", map[string]any{"recurso": "reserva", "id": reservaID})
	}
	if err != nil {
		return Reserva{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT id FROM productos WHERE id = $1 FOR UPDATE`, productoID); err != nil {
		return Reserva{}, err
	}
	var estado, cantidad string
	err = tx.QueryRow(ctx, `SELECT estado, cantidad::text FROM reservas WHERE id = $1 FOR UPDATE`, reservaID).Scan(&estado, &cantidad)
	if err != nil {
		return Reserva{}, err
	}
	if estado == "activa" {
		if _, err := tx.Exec(ctx, `UPDATE productos SET stock = stock + $2::numeric, actualizado_en = now() WHERE id = $1`, productoID, cantidad); err != nil {
			return Reserva{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE reservas SET estado = 'compensada', compensado_en = now() WHERE id = $1`, reservaID); err != nil {
			return Reserva{}, err
		}
	}
	var out Reserva
	if err := tx.QueryRow(ctx, `
		SELECT id::text, producto_id::text, productor_id::text, nombre, unidad, precio_centavos, cantidad::text, referencia, estado, creado_en
		FROM reservas WHERE id = $1`, reservaID,
	).Scan(&out.ID, &out.ProductoID, &out.ProductorID, &out.Nombre, &out.Unidad, &out.PrecioCentavos, &out.Cantidad, &out.Referencia, &out.Estado, &out.CreadoEn); err != nil {
		return Reserva{}, err
	}
	out.Cantidad = mustFixed(out.Cantidad)
	out.CreadoEn = out.CreadoEn.UTC()
	if err := tx.Commit(ctx); err != nil {
		return Reserva{}, err
	}
	return out, nil
}

func parseCantidadLibre(raw string) (decimal.Decimal, *Error) {
	return ParseCantidad(raw, "kg", "cantidad", false)
}

func mustFixed(raw string) string {
	d, err := decimal.NewFromString(raw)
	if err != nil {
		return raw
	}
	return FormatoCantidad(d)
}

func escaparLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProducto(row scanner) (Producto, error) {
	var p Producto
	var stock string
	if err := row.Scan(
		&p.ID, &p.Productor.ID, &p.Productor.Nombre, &p.Productor.Distrito,
		&p.Nombre, &p.Categoria, &p.Unidad, &p.PrecioCentavos, &stock, &p.Activo, &p.Descripcion,
		&p.CreadoEn, &p.ActualizadoEn,
	); err != nil {
		return p, err
	}
	p.Stock = mustFixed(stock)
	p.CreadoEn = p.CreadoEn.UTC()
	p.ActualizadoEn = p.ActualizadoEn.UTC()
	return p, nil
}
