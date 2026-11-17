CREATE TABLE IF NOT EXISTS esquema (
    version INT PRIMARY KEY,
    aplicado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS clientes (
    id UUID PRIMARY KEY,
    nombre TEXT NOT NULL,
    telefono TEXT NOT NULL,
    distrito TEXT NOT NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT clientes_telefono_chk CHECK (telefono ~ '^\+51[0-9]{9}$')
);

CREATE TABLE IF NOT EXISTS pedidos (
    id UUID PRIMARY KEY,
    cliente_id UUID NOT NULL REFERENCES clientes (id),
    estado TEXT NOT NULL,
    total_centavos INT NOT NULL,
    notas TEXT NOT NULL DEFAULT '',
    clave_idempotencia TEXT UNIQUE,
    cuerpo_hash TEXT,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pedidos_estado_chk CHECK (
        estado IN ('reservando', 'pendiente', 'confirmado', 'cancelado', 'compensacion_incompleta')
    ),
    CONSTRAINT pedidos_total_chk CHECK (total_centavos >= 0)
);

CREATE INDEX IF NOT EXISTS pedidos_cliente_idx ON pedidos (cliente_id);
CREATE INDEX IF NOT EXISTS pedidos_estado_idx ON pedidos (estado, creado_en DESC);

CREATE TABLE IF NOT EXISTS lineas (
    id UUID PRIMARY KEY,
    pedido_id UUID NOT NULL REFERENCES pedidos (id) ON DELETE CASCADE,
    producto_id UUID NOT NULL,
    productor_id UUID NOT NULL,
    reserva_id UUID NOT NULL,
    nombre_producto TEXT NOT NULL,
    unidad TEXT NOT NULL,
    precio_unitario_centavos INT NOT NULL,
    cantidad NUMERIC(12,3) NOT NULL,
    subtotal_centavos INT NOT NULL,
    UNIQUE (pedido_id, producto_id)
);
