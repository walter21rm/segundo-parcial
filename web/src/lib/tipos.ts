export type Productor = {
  id: string;
  nombre: string;
  distrito: string;
  telefono: string;
  descripcion: string;
  productos_activos: number;
  creado_en: string;
};

export type Producto = {
  id: string;
  productor: { id: string; nombre: string; distrito: string };
  nombre: string;
  categoria: string;
  unidad: string;
  precio_centavos: number;
  stock: string;
  activo: boolean;
  descripcion: string;
};

export type Cliente = {
  id: string;
  nombre: string;
  telefono: string;
  distrito: string;
};

export type LineaPedido = {
  id: string;
  producto_id: string;
  productor_id: string;
  nombre_producto: string;
  unidad: string;
  precio_unitario_centavos: number;
  cantidad: string;
  subtotal_centavos: number;
};

export type Pedido = {
  id: string;
  cliente: Cliente;
  estado: string;
  total_centavos: number;
  notas: string;
  lineas: LineaPedido[];
  creado_en: string;
  actualizado_en: string;
};

export type Pagina<T> = {
  datos: T[];
  paginacion: { limite: number; desplazamiento: number; total: number };
};

export type Resultado<T> =
  | { ok: true; datos: T }
  | { ok: false; mensaje: string };
