package catalogo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Servicio struct {
	store   *Store
	version string
}

func NuevoServicio(store *Store, version string) http.Handler {
	s := &Servicio{store: store, version: version}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /salud", s.salud)
	mux.HandleFunc("GET /listo", s.listo)
	mux.HandleFunc("GET /version", s.versionHandler)
	mux.HandleFunc("GET /v1/productores", s.listarProductores)
	mux.HandleFunc("POST /v1/productores", s.crearProductor)
	mux.HandleFunc("GET /v1/productores/{id}", s.obtenerProductor)
	mux.HandleFunc("PATCH /v1/productores/{id}", s.actualizarProductor)
	mux.HandleFunc("DELETE /v1/productores/{id}", s.eliminarProductor)
	mux.HandleFunc("GET /v1/productos", s.listarProductos)
	mux.HandleFunc("POST /v1/productos", s.crearProducto)
	mux.HandleFunc("GET /v1/productos/{id}", s.obtenerProducto)
	mux.HandleFunc("PATCH /v1/productos/{id}", s.actualizarProducto)
	mux.HandleFunc("DELETE /v1/productos/{id}", s.desactivarProducto)
	mux.HandleFunc("POST /v1/productos/{id}/reservas", s.reservar)
	mux.HandleFunc("POST /v1/reservas/{id}/compensar", s.compensar)
	return s.conRequestID(mux)
}

func (s *Servicio) salud(w http.ResponseWriter, r *http.Request) {
	escribirJSON(w, http.StatusOK, map[string]string{"estado": "ok", "servicio": "catalogo"})
}

func (s *Servicio) listo(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		escribirError(w, r, api(http.StatusServiceUnavailable, "NO_LISTO", "La base de datos no responde.", nil))
		return
	}
	escribirJSON(w, http.StatusOK, map[string]string{"estado": "listo", "servicio": "catalogo"})
}

func (s *Servicio) versionHandler(w http.ResponseWriter, r *http.Request) {
	escribirJSON(w, http.StatusOK, map[string]string{"servicio": "catalogo", "version": s.version})
}

func (s *Servicio) listarProductores(w http.ResponseWriter, r *http.Request) {
	limite, desplazamiento, err := leerPaginacion(r)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	pagina, errQ := s.store.ListarProductores(r.Context(), limite, desplazamiento)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, pagina)
}

func (s *Servicio) crearProductor(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Nombre      string `json:"nombre"`
		Distrito    string `json:"distrito"`
		Telefono    string `json:"telefono"`
		Descripcion string `json:"descripcion"`
	}
	if err := leerJSON(w, r, &body); err != nil {
		escribirError(w, r, err)
		return
	}
	nombre, err := Texto(body.Nombre, "nombre", 2, 120)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	distrito, err := Texto(body.Distrito, "distrito", 2, 80)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	telefono, err := ValidarTelefono(body.Telefono)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	desc, err := Texto(body.Descripcion, "descripcion", 0, 500)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	p, errQ := s.store.CrearProductor(r.Context(), NuevoProductor{Nombre: nombre, Distrito: distrito, Telefono: telefono, Descripcion: desc})
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusCreated, p)
}

func (s *Servicio) obtenerProductor(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	p, errQ := s.store.ObtenerProductor(r.Context(), id)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, p)
}

func (s *Servicio) actualizarProductor(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	raw, err := leerObjeto(w, r)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	if len(raw) == 0 {
		escribirError(w, r, solicitud("No hay campos para actualizar.", nil))
		return
	}
	var parche ParcheProductor
	if v, ok := raw["nombre"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			escribirError(w, r, solicitud("El campo nombre debe ser texto.", map[string]any{"campo": "nombre"}))
			return
		}
		nombre, err := Texto(s, "nombre", 2, 120)
		if err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Nombre = &nombre
	}
	if v, ok := raw["distrito"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			escribirError(w, r, solicitud("El campo distrito debe ser texto.", map[string]any{"campo": "distrito"}))
			return
		}
		distrito, err := Texto(s, "distrito", 2, 80)
		if err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Distrito = &distrito
	}
	if v, ok := raw["telefono"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			escribirError(w, r, solicitud("El campo telefono debe ser texto.", map[string]any{"campo": "telefono"}))
			return
		}
		telefono, err := ValidarTelefono(s)
		if err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Telefono = &telefono
	}
	if v, ok := raw["descripcion"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			escribirError(w, r, solicitud("El campo descripcion debe ser texto.", map[string]any{"campo": "descripcion"}))
			return
		}
		desc, err := Texto(s, "descripcion", 0, 500)
		if err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Descripcion = &desc
	}
	for k := range raw {
		switch k {
		case "nombre", "distrito", "telefono", "descripcion":
		default:
			escribirError(w, r, solicitud("Hay un campo que este recurso no acepta.", map[string]any{"campo": k}))
			return
		}
	}
	p, errQ := s.store.ActualizarProductor(r.Context(), id, parche)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, p)
}

func (s *Servicio) eliminarProductor(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	if errQ := s.store.EliminarProductor(r.Context(), id); errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Servicio) listarProductos(w http.ResponseWriter, r *http.Request) {
	limite, desplazamiento, err := leerPaginacion(r)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	f := FiltroProductos{Limite: limite, Desplazamiento: desplazamiento}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) > 80 {
		escribirError(w, r, solicitud("La búsqueda admite hasta 80 caracteres.", map[string]any{"campo": "q"}))
		return
	}
	f.Q = q
	if c := r.URL.Query().Get("categoria"); c != "" {
		if err := ValidarCategoria(c); err != nil {
			escribirError(w, r, err)
			return
		}
		f.Categoria = c
	}
	if id := r.URL.Query().Get("productor_id"); id != "" {
		norm, err := NormalizarID(id)
		if err != nil {
			err.Detalles["campo"] = "productor_id"
			escribirError(w, r, err)
			return
		}
		f.ProductorID = norm
	}
	if a := r.URL.Query().Get("activo"); a != "" {
		if a != "true" && a != "false" {
			escribirError(w, r, solicitud("activo debe ser true o false.", map[string]any{"campo": "activo"}))
			return
		}
		v := a == "true"
		f.Activo = &v
	}
	pagina, errQ := s.store.ListarProductos(r.Context(), f)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, pagina)
}

func (s *Servicio) crearProducto(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProductorID    string `json:"productor_id"`
		Nombre         string `json:"nombre"`
		Categoria      string `json:"categoria"`
		Unidad         string `json:"unidad"`
		PrecioCentavos int    `json:"precio_centavos"`
		Stock          string `json:"stock"`
		Descripcion    string `json:"descripcion"`
	}
	if err := leerJSON(w, r, &body); err != nil {
		escribirError(w, r, err)
		return
	}
	productorID, err := NormalizarID(body.ProductorID)
	if err != nil {
		err.Detalles["campo"] = "productor_id"
		escribirError(w, r, err)
		return
	}
	nombre, err := Texto(body.Nombre, "nombre", 2, 120)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	if err := ValidarCategoria(body.Categoria); err != nil {
		escribirError(w, r, err)
		return
	}
	if err := ValidarUnidad(body.Unidad); err != nil {
		escribirError(w, r, err)
		return
	}
	if err := ValidarPrecio(body.PrecioCentavos); err != nil {
		escribirError(w, r, err)
		return
	}
	stock, err := ParseCantidad(body.Stock, body.Unidad, "stock", true)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	desc, err := Texto(body.Descripcion, "descripcion", 0, 500)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	p, errQ := s.store.CrearProducto(r.Context(), NuevoProducto{
		ProductorID: productorID, Nombre: nombre, Categoria: body.Categoria, Unidad: body.Unidad,
		PrecioCentavos: body.PrecioCentavos, Stock: stock, Descripcion: desc,
	})
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusCreated, p)
}

func (s *Servicio) obtenerProducto(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	p, errQ := s.store.ObtenerProducto(r.Context(), id)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, p)
}

func (s *Servicio) actualizarProducto(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	raw, err := leerObjeto(w, r)
	if err != nil {
		escribirError(w, r, err)
		return
	}
	if len(raw) == 0 {
		escribirError(w, r, solicitud("No hay campos para actualizar.", nil))
		return
	}
	var parche ParcheProducto
	unidadParaStock := "kg"
	if v, ok := raw["unidad"]; ok {
		var s string
		if e := json.Unmarshal(v, &s); e != nil {
			escribirError(w, r, solicitud("El campo unidad debe ser texto.", map[string]any{"campo": "unidad"}))
			return
		}
		if err := ValidarUnidad(s); err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Unidad = &s
		unidadParaStock = s
	}
	if v, ok := raw["nombre"]; ok {
		var s string
		if e := json.Unmarshal(v, &s); e != nil {
			escribirError(w, r, solicitud("El campo nombre debe ser texto.", map[string]any{"campo": "nombre"}))
			return
		}
		nombre, err := Texto(s, "nombre", 2, 120)
		if err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Nombre = &nombre
	}
	if v, ok := raw["categoria"]; ok {
		var s string
		if e := json.Unmarshal(v, &s); e != nil {
			escribirError(w, r, solicitud("El campo categoria debe ser texto.", map[string]any{"campo": "categoria"}))
			return
		}
		if err := ValidarCategoria(s); err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Categoria = &s
	}
	if v, ok := raw["descripcion"]; ok {
		var s string
		if e := json.Unmarshal(v, &s); e != nil {
			escribirError(w, r, solicitud("El campo descripcion debe ser texto.", map[string]any{"campo": "descripcion"}))
			return
		}
		desc, err := Texto(s, "descripcion", 0, 500)
		if err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Descripcion = &desc
	}
	if v, ok := raw["precio_centavos"]; ok {
		var n int
		if e := json.Unmarshal(v, &n); e != nil {
			escribirError(w, r, solicitud("El precio debe ser un entero.", map[string]any{"campo": "precio_centavos"}))
			return
		}
		if err := ValidarPrecio(n); err != nil {
			escribirError(w, r, err)
			return
		}
		parche.PrecioCentavos = &n
	}
	if v, ok := raw["stock"]; ok {
		var s string
		if e := json.Unmarshal(v, &s); e != nil {
			escribirError(w, r, solicitud("El stock debe ser un texto decimal.", map[string]any{"campo": "stock"}))
			return
		}
		stock, err := ParseCantidad(s, unidadParaStock, "stock", true)
		if err != nil {
			escribirError(w, r, err)
			return
		}
		parche.Stock = &stock
	}
	if v, ok := raw["activo"]; ok {
		var b bool
		if e := json.Unmarshal(v, &b); e != nil {
			escribirError(w, r, solicitud("activo debe ser true o false.", map[string]any{"campo": "activo"}))
			return
		}
		parche.Activo = &b
	}
	for k := range raw {
		switch k {
		case "nombre", "categoria", "unidad", "descripcion", "precio_centavos", "stock", "activo":
		default:
			escribirError(w, r, solicitud("Hay un campo que este recurso no acepta.", map[string]any{"campo": k}))
			return
		}
	}
	p, errQ := s.store.ActualizarProducto(r.Context(), id, parche)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, p)
}

func (s *Servicio) desactivarProducto(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	p, errQ := s.store.DesactivarProducto(r.Context(), id)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, p)
}

func (s *Servicio) reservar(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	var body struct {
		Cantidad   string `json:"cantidad"`
		Referencia string `json:"referencia"`
	}
	if err := leerJSON(w, r, &body); err != nil {
		escribirError(w, r, err)
		return
	}
	if err := ValidarReferencia(body.Referencia); err != nil {
		escribirError(w, r, err)
		return
	}
	reserva, status, errQ := s.store.Reservar(r.Context(), id, body.Cantidad, body.Referencia)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, status, reserva)
}

func (s *Servicio) compensar(w http.ResponseWriter, r *http.Request) {
	id, err := NormalizarID(r.PathValue("id"))
	if err != nil {
		escribirError(w, r, err)
		return
	}
	if r.Body != nil {
		buf, _ := io.ReadAll(io.LimitReader(r.Body, 32))
		if len(strings.TrimSpace(string(buf))) > 0 {
			escribirError(w, r, solicitud("Compensar no recibe cuerpo.", nil))
			return
		}
	}
	reserva, errQ := s.store.Compensar(r.Context(), id)
	if errQ != nil {
		escribirFallo(w, r, errQ)
		return
	}
	escribirJSON(w, http.StatusOK, reserva)
}

func leerPaginacion(r *http.Request) (int, int, *Error) {
	limite := 20
	desplazamiento := 0
	if v := r.URL.Query().Get("limite"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return 0, 0, solicitud("limite debe ser un entero entre 1 y 100.", map[string]any{"campo": "limite"})
		}
		limite = n
	}
	if v := r.URL.Query().Get("desplazamiento"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 10000 {
			return 0, 0, solicitud("desplazamiento debe ser un entero entre 0 y 10000.", map[string]any{"campo": "desplazamiento"})
		}
		desplazamiento = n
	}
	return limite, desplazamiento, nil
}

func leerJSON(w http.ResponseWriter, r *http.Request, dest any) *Error {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		return solicitud("El cuerpo debe ser application/json.", nil)
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			return solicitud("El cuerpo supera 1 MiB.", nil)
		}
		return solicitud("El JSON no es válido.", nil)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return solicitud("El cuerpo tiene más de un valor JSON.", nil)
	}
	return nil
}

func leerObjeto(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, *Error) {
	var raw map[string]json.RawMessage
	if err := leerJSON(w, r, &raw); err != nil {
		return nil, err
	}
	for k, v := range raw {
		if string(v) == "null" {
			return nil, solicitud("No se acepta null.", map[string]any{"campo": k})
		}
	}
	return raw, nil
}

func escribirJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func escribirError(w http.ResponseWriter, r *http.Request, err *Error) {
	if err.Detalles == nil {
		err.Detalles = map[string]any{}
	}
	escribirJSON(w, err.HTTP, map[string]any{
		"error": map[string]any{
			"codigo":     err.Codigo,
			"mensaje":    err.Mensaje,
			"request_id": requestID(r),
			"detalles":   err.Detalles,
		},
	})
}

func escribirFallo(w http.ResponseWriter, r *http.Request, err error) {
	var ae *Error
	if errors.As(err, &ae) {
		escribirError(w, r, ae)
		return
	}
	slog.Error("error interno", "err", err, "request_id", requestID(r))
	escribirError(w, r, api(http.StatusInternalServerError, "ERROR_INTERNO", "Error interno.", nil))
}

type ctxKey int

const requestIDKey ctxKey = 1

func requestID(r *http.Request) string {
	v, _ := r.Context().Value(requestIDKey).(string)
	return v
}

func contextWithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func (s *Servicio) conRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if _, err := NormalizarID(id); err != nil {
			id = nuevoID()
		} else {
			id, _ = NormalizarID(id)
		}
		w.Header().Set("X-Request-Id", id)
		ctx := r.Context()
		ctx = contextWithID(ctx, id)
		inicio := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))
		slog.Info("solicitud", "servicio", "catalogo", "metodo", r.Method, "ruta", r.URL.Path, "request_id", id, "ms", time.Since(inicio).Milliseconds())
	})
}
