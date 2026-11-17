CREATE TABLE IF NOT EXISTS esquema (
    version INT PRIMARY KEY,
    aplicado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS productores (
    id UUID PRIMARY KEY,
    nombre TEXT NOT NULL,
    distrito TEXT NOT NULL,
    telefono TEXT NOT NULL,
    descripcion TEXT NOT NULL DEFAULT '',
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT productores_telefono_chk CHECK (telefono ~ '^\+51[0-9]{9}$')
);

CREATE TABLE IF NOT EXISTS productos (
    id UUID PRIMARY KEY,
    productor_id UUID NOT NULL REFERENCES productores (id),
    nombre TEXT NOT NULL,
    categoria TEXT NOT NULL,
    unidad TEXT NOT NULL,
    precio_centavos INT NOT NULL,
    stock NUMERIC(12,3) NOT NULL,
    activo BOOLEAN NOT NULL DEFAULT TRUE,
    descripcion TEXT NOT NULL DEFAULT '',
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT productos_categoria_chk CHECK (categoria IN ('frutas', 'verduras', 'lacteos', 'granos', 'miel', 'otros')),
    CONSTRAINT productos_unidad_chk CHECK (unidad IN ('kg', 'unidad', 'litro', 'atado')),
    CONSTRAINT productos_precio_chk CHECK (precio_centavos > 0 AND precio_centavos <= 100000000),
    CONSTRAINT productos_stock_chk CHECK (stock >= 0 AND stock <= 1000000)
);

CREATE INDEX IF NOT EXISTS productos_productor_idx ON productos (productor_id);
CREATE INDEX IF NOT EXISTS productos_categoria_idx ON productos (categoria);
CREATE INDEX IF NOT EXISTS productos_nombre_idx ON productos (nombre);

CREATE TABLE IF NOT EXISTS reservas (
    id UUID PRIMARY KEY,
    producto_id UUID NOT NULL REFERENCES productos (id),
    productor_id UUID NOT NULL,
    nombre TEXT NOT NULL,
    unidad TEXT NOT NULL,
    precio_centavos INT NOT NULL,
    cantidad NUMERIC(12,3) NOT NULL,
    referencia TEXT NOT NULL UNIQUE,
    estado TEXT NOT NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    compensado_en TIMESTAMPTZ,
    CONSTRAINT reservas_estado_chk CHECK (estado IN ('activa', 'compensada')),
    CONSTRAINT reservas_cantidad_chk CHECK (cantidad > 0 AND cantidad <= 10000)
);

CREATE INDEX IF NOT EXISTS reservas_producto_idx ON reservas (producto_id);
