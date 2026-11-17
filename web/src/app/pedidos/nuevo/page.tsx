import { Aviso } from "@/components/aviso";
import { FormularioPedido } from "@/components/formulario-pedido";
import { listarClientes } from "@/lib/api";

export default async function NuevoPedido() {
  const clientes = await listarClientes();
  if (!clientes.ok) return <Aviso titulo="No se pudo preparar el pedido" mensaje={clientes.mensaje} />;
  return <FormularioPedido clientes={clientes.datos.datos} />;
}
