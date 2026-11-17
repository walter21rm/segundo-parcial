# Especificación cerrada — Feria del Productor

Versión 1.0.0. Este documento es el contrato. Un cambio de campo, código de error, estado o puerto es una versión nueva, no un ajuste.

## Qué es cada servicio

Hay dos servicios independientes. No comparten base de datos, tablas ni proceso.

| Servicio | Lenguaje | Base | Puerto | Responsabilidad |
| --- | --- | --- | --- | --- |
| Catálogo | Go 1.22 | PostgreSQL 16 propio | 8091 | Productores, productos, stock y reservas |
| Pedidos | Python 3.12 y FastAPI | PostgreSQL 16 propio, otra instancia | 8092 | Clientes y pedidos. Reserva stock llamando al catálogo |
| Web | Next.js 15 y TypeScript | ninguna | 3847 | Arma la pantalla y habla con los dos servicios desde el servidor |

El navegador no llama a Go ni a Python. Solo llama a la web. La web llama a `CATALOGO_URL` y `PEDIDOS_URL`.

## Fuera de esta versión

No hay autenticación, pagos, notificaciones, edición de un pedido ya creado, borrado de clientes, HPA, service mesh, RDS, ni network policies. El precio de una reserva activa no se recalcula si el puesto cambia el precio después. HTTPS en el balanceador queda para cuando exista un certificado ACM: la versión 1 publica HTTP en el puerto 80.

## Reglas comunes

- Identificadores UUID en minúsculas. Se acepta cualquier caja y se normaliza.
- Fechas en UTC, formato `2006-01-17T20:00:00Z`.
- Dinero en céntimos de sol, entero. `890` es S/ 8.90.
- Cantidades y stock viajan como texto decimal, nunca como número JSON. En las respuestas siempre hay 3 decimales (`1.500`).
- El subtotal es `round_half_up(precio_centavos × cantidad)` al céntimo. Ejemplo cerrado: 890 × 1.5 = 1335. 100 × 0.335 = 34. 1 × 0.5 = 1.
- `kg` y `litro` admiten hasta 3 decimales. `unidad` y `atado` solo enteros (`2` y `2.000` valen, `2.500` no).
- Una reserva exige cantidad > 0 y ≤ 10000. El stock admite 0 y llega hasta 1000000.
- Teléfono: `+51` y 9 dígitos.
- Textos sin caracteres de control. Nombre 2–120. Distrito 2–80. Descripción y notas 0–500.
- Categorías: `frutas`, `verduras`, `lacteos`, `granos`, `miel`, `otros`.
- Unidades: `kg`, `unidad`, `litro`, `atado`.
- Paginación: `limite` 1–100, por defecto 20. `desplazamiento` 0–10000, por defecto 0.
- Cuerpo JSON máximo 1 MiB. Los objetos rechazan campos desconocidos y `null`.
- Cada respuesta lleva `X-Request-Id`. Si el cliente no manda un UUID, el servicio genera uno.
- El error siempre tiene esta forma:

```json
{
  "error": {
    "codigo": "STOCK_INSUFICIENTE",
    "mensaje": "No hay stock suficiente.",
    "request_id": "…",
    "detalles": {}
  }
}
```

- `GET /salud` responde 200 si el proceso vive. `GET /listo` responde 200 solo si la base contesta, si no 503 `NO_LISTO`. `GET /version` devuelve el nombre y `1.0.0`.
- No hay `/docs` ni `/openapi.json`. El contrato es este archivo.
- El esquema se crea al arrancar si falta y queda marcado como versión 1. Si la tabla `esquema` tiene otro número, el proceso no arranca.

## Catálogo

### Productor

`id`, `nombre`, `distrito`, `telefono`, `descripcion`, `productos_activos`, `creado_en`.

- `GET /v1/productores`
- `POST /v1/productores` → 201
- `GET /v1/productores/{id}`
- `PATCH /v1/productores/{id}` con al menos un campo
- `DELETE /v1/productores/{id}` → 204. Si tiene productos, 409 `PRODUCTOR_CON_PRODUCTOS`. No se borran productos en cascada.

### Producto

Incluye el productor embebido (`id`, `nombre`, `distrito`), `categoria`, `unidad`, `precio_centavos`, `stock`, `activo`, `descripcion`, `creado_en`, `actualizado_en`.

- `GET /v1/productos` filtra por `q` (nombre, sin distinguir mayúsculas, hasta 80 caracteres), `categoria`, `productor_id` y `activo` (`true` o `false`; si se omite, entran todos). Orden por nombre.
- `POST /v1/productos` → 201. Nace activo. El productor tiene que existir.
- `GET /v1/productos/{id}` también devuelve inactivos.
- `PATCH /v1/productos/{id}`. No se cambia de productor. Si la unidad nueva no admite el stock actual, 400.
- `DELETE /v1/productos/{id}` no borra: deja `activo` en false y responde 200. Es idempotente.

### Reserva

`POST /v1/productos/{id}/reservas` con `{ "cantidad", "referencia" }`.

La referencia mide entre 8 y 120 caracteres y solo usa letras, números y `: . _ -`.

- 201 la primera vez, con el snapshot del momento: nombre, productor, unidad, precio y cantidad.
- 200 si la misma referencia ya está `activa` con el mismo producto y la misma cantidad. Devuelve el snapshot original, aunque el precio haya cambiado.
- 200 si estaba `compensada` con el mismo producto y cantidad: vuelve a descontar stock y refresca el snapshot al precio vigente.
- 409 `REFERENCIA_CONFLICTO` si la referencia ya existe con otro producto o cantidad.
- 409 `STOCK_INSUFICIENTE` con `stock` y `solicitado`.
- 409 `PRODUCTO_INACTIVO`.
- 404 `NO_ENCONTRADO` si no hay producto.

`POST /v1/reservas/{id}/compensar` no lleva cuerpo. Devuelve el stock y deja la reserva `compensada`. Repetirla responde 200 sin volver a sumar.

El descuento y la compensación bloquean primero la fila del producto.

### Semilla

Solo inserta ids que no existen. No pisa stock ni precios ya guardados. Se ejecuta si `SEED=true`.

| Id | Qué es |
| --- | --- |
| `11111111-1111-4111-8111-111111111111` | Huerta Santa Rosa, Santiago de Surco |
| `22222222-2222-4222-8222-222222222222` | Fundo La Molina, La Molina |
| `33333333-3333-4333-8333-333333333333` | Quesería Barranco, Barranco |
| `44444444-4444-4444-8444-444444444444` | Apiario Pachacámac, Pachacámac |
| `55555555-5555-4555-8555-555555555555` | Granos del Valle, Lurín |

Productos `10000000-0000-4000-8000-000000000001` a `…000e`: espinaca, tomate, lechuga, albahaca, palta (890 céntimos, stock inicial 18.500 kg), fresa, plátano, limón, queso, yogur, miel de eucalipto, miel de algarrobo, quinua y maíz morado. Teléfonos de productores `+51941111001` a `+51941111005`.

## Pedidos

### Cliente

`POST /v1/clientes` → 201. `GET /v1/clientes`. `GET /v1/clientes/{id}`. No hay edición ni borrado. Si no existe, 404 `CLIENTE_NO_ENCONTRADO`.

Semilla, con `SEED=true` y sin pisar: Rosa Quispe (`aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1`, Surquillo), Luis Mendoza (`…aaa2`, Miraflores), Elena Vargas (`…aaa3`, San Borja). Teléfonos `+51942222001` a `+51942222003`.

### Pedido

Estados: `reservando`, `pendiente`, `confirmado`, `cancelado`, `compensacion_incompleta`.

El listado sin `estado` solo muestra `pendiente`, `confirmado` y `cancelado`. Con `?estado=` se puede pedir cualquiera de los cinco.

`POST /v1/pedidos` con `cliente_id`, `notas` y `lineas` (de 1 a 20, sin repetir producto). Cada línea es `producto_id` y `cantidad`.

La cabecera `Idempotency-Key` es opcional, de 8 a 80 caracteres (`letras, números, . _ : -`). La web siempre la envía.

1. Si la clave ya tiene un pedido `pendiente`, `confirmado` o `cancelado` con el mismo cuerpo, responde 200 y `Idempotent-Replayed: true`.
2. Si la clave existe con otro cuerpo, 409 `IDEMPOTENCIA_CONFLICTO`.
3. Si alguien más está usando esa clave o ese pedido, 409 `IDEMPOTENCIA_EN_CURSO` o `PEDIDO_EN_CURSO`.
4. Si no, crea el pedido en `reservando` y reserva cada línea en orden. La referencia de catálogo es `ped:{pedido_id}:prod:{producto_id}`.
5. Si una reserva falla, compensa en orden inverso las que sí se guardaron. Si la compensación termina, borra el pedido y devuelve el error del catálogo (`STOCK_INSUFICIENTE`, `PRODUCTO_INACTIVO`, `PRODUCTO_NO_ENCONTRADO`, `SOLICITUD_INVALIDA` o `CATALOGO_NO_DISPONIBLE`).
6. Si la compensación no termina, el pedido queda en `compensacion_incompleta` y responde 500 `COMPENSACION_INCOMPLETA`. Un reintento con la misma clave compensa otra vez y, si ahora puede, sigue el alta.
7. Si el proceso muere en `reservando`, la misma clave reanuda las líneas que faltan. La reserva repetida es idempotente.
8. Al completar, el estado pasa a `pendiente`, el total es la suma de subtotales y la respuesta es 201.

`POST /v1/pedidos/{id}/confirmar` no lleva cuerpo. Solo desde `pendiente`. Repetirlo en `confirmado` responde 200. Otro estado: 409 `ESTADO_INVALIDO`. No toca el stock.

`POST /v1/pedidos/{id}/cancelar` no lleva cuerpo. Desde `pendiente`, `confirmado`, `reservando` o `compensacion_incompleta` compensa cada línea y deja `cancelado`. Repetirlo responde 200. Si una compensación falla, no cambia un pedido público a cancelado y responde 500 `COMPENSACION_INCOMPLETA`.

El cliente HTTP reintenta una vez la reserva y la compensación si no hubo respuesta (corte o tiempo de espera). No reintenta un 4xx. El tiempo de espera es `CATALOGO_TIMEOUT_SEGUNDOS`, por defecto 3.

## Web

| Ruta | Qué hace |
| --- | --- |
| `/` | Productos activos, búsqueda y categoría. Agregar arma el pedido en el navegador (`localStorage`, clave `feria.carrito.v1`) |
| `/productores` y `/productores/{id}` | Puestos y su mercadería |
| `/pedidos/nuevo` | Elige un cliente existente o crea uno, y registra el pedido |
| `/pedidos` y `/pedidos/{id}` | Lista y detalle. Confirmar o cancelar |
| `/api/salud` | 200 siempre que la web viva. El cuerpo dice si catálogo y pedidos están `ok` o `caido` |

La web vacía, la caída de un servicio y el pedido sin líneas tienen su propio texto. No hay página en blanco.

## Configuración

`DATABASE_URL` gana. Si no está, la URL se arma con `PGHOST` (127.0.0.1), `PGPORT` (5432), `PGUSER`, `PGPASSWORD`, `PGDATABASE` y `sslmode=disable`. Ese modo sin TLS es el de la versión 1: las bases viven en el mismo clúster que los servicios.

| Variable | Dónde |
| --- | --- |
| `PORT` | 8091, 8092 o 3847 |
| `SEED` | `true` inserta la semilla que falte |
| `CATALOGO_URL` | pedidos y web |
| `PEDIDOS_URL` | web |
| `CATALOGO_TIMEOUT_SEGUNDOS` | pedidos |
| `VERSION` | ambos servicios, por defecto 1.0.0 |

Las contraseñas `catalogo_dev` y `pedidos_dev` son solo para la máquina local. En Kubernetes el secreto `feria-db` trae `cambiar-en-aws` a propósito, para reemplazarlo antes de exponer el clúster.

## Kubernetes en AWS

Clúster EKS 1.31 estándar, no Auto Mode, para poder usar las anotaciones del AWS Load Balancer Controller. El archivo `deploy/aws/cluster.yaml` lo describe: dos nodos `t3.medium` en subredes privadas, OIDC y el addon `aws-ebs-csi-driver`.

Dentro del namespace `feria`:

- Dos Deployment de PostgreSQL 16.6, uno por servicio, cada uno con su PVC de 5 Gi en la clase `feria-gp3` (`ebs.csi.aws.com`, tipo gp3). `PGDATA` es `/var/lib/postgresql/data/pgdata`. El proceso corre con uid 999.
- Catálogo, pedidos y web. Una réplica cada uno. Sondas en `/salud`, `/listo` y, para la web, `/api/salud`.
- La web se publica en el Service puerto 80 hacia el contenedor 3847.
- El overlay `deploy/k8s/overlays/aws` agrega un Ingress `alb`, de cara a internet, `target-type: ip`, HTTP 80, healthcheck `/api/salud`. No exige dominio ni certificado.
- El overlay `local` quita la clase de almacenamiento para usar la del clúster de prueba.
- Catálogo y pedidos se hablan por DNS interno: `http://catalogo:8091`. No salen por el balanceador.

El controlador del balanceador se instala con Helm y Pod Identity, como documenta AWS Load Balancer Controller, antes de aplicar el overlay. `deploy/aws/subir.sh` crea los repositorios ECR `feria-catalogo`, `feria-pedidos` y `feria-web`, publica el tag `1.0.0` y aplica el overlay.

No se crea el clúster desde esta máquina: un EKS nuevo tiene costo y exige credenciales de la cuenta.
