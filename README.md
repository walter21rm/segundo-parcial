# Segundo parcial — Feria del Productor

Dos servicios independientes y una web que los consulta. El catálogo está en Go y los pedidos en Python. Cada uno tiene su propia base PostgreSQL. La web, en Next.js, arma el puesto y el pedido sin que el navegador hable con los servicios.

El contrato cerrado está en [docs/ESPECIFICACION.md](docs/ESPECIFICACION.md). La versión 1 no se aparta de ese archivo.

## Cómo se usa

En el puesto se ven los productos de la mañana. Agregar los junta en un pedido local. Al registrarlo, pedidos reserva el stock en el catálogo y guarda el precio de ese momento. Confirmar lo deja apartado. Cancelar devuelve el stock.

Rosa Quispe, Luis Mendoza y Elena Vargas ya están como clientes. Hay cinco puestos de Lima: Surco, La Molina, Barranco, Pachacámac y Lurín.

## Correr en esta máquina

Hace falta Go 1.22, Python 3.12, Node 22 y PostgreSQL 16.

```bash
sudo service postgresql start
# usuarios catalogo / pedidos y bases catalogo y pedidos, como en la especificación
bash scripts/dev.sh
```

La web queda en el puerto 3847. Catálogo en 8091 y pedidos en 8092.

Con Docker, desde la raíz:

```bash
docker compose up --build
```

## Pruebas

```bash
cd servicios/catalogo
DATABASE_URL='postgres://catalogo:catalogo_dev@127.0.0.1:5432/catalogo_test?sslmode=disable' go test ./...

cd servicios/pedidos
DATABASE_URL='postgres://pedidos:pedidos_dev@127.0.0.1:5432/pedidos_test?sslmode=disable' .venv/bin/pytest -q
```

El caso cerrado de dinero es 890 céntimos por 1.5 kg = 1335 céntimos.

## Kubernetes en AWS

1. Crear el clúster EKS 1.35 con `deploy/aws/cluster.yaml` (`eksctl create cluster -f deploy/aws/cluster.yaml`). No es EKS Auto Mode. Los nodos son `t3.small` porque esta cuenta solo lanza tipos del nivel gratuito. El Ingress usa el AWS Load Balancer Controller.
2. Instalar ese controlador con Helm y Pod Identity, y el addon EBS CSI que ya pide el manifiesto del clúster.
3. Cambiar las claves `catalogo-password` y `pedidos-password` del secreto `feria-db` antes de exponerlo. El valor del repositorio es un marcador.
4. Publicar imágenes y aplicar el overlay:

```bash
export AWS_REGION=us-east-1
export CLUSTER_NAME=feria
bash deploy/aws/subir.sh
```

El balanceador solo abre la web. Catálogo y pedidos se quedan dentro del namespace `feria`, cada uno contra su Postgres.

El clúster `feria` en `us-east-1` publica el puesto en:

http://k8s-feria-web-2be4ec8368-557112646.us-east-1.elb.amazonaws.com

Para dejar de pagar el plano de control, el NAT, el balanceador y los nodos:

```bash
kubectl delete -k deploy/k8s/overlays/aws
eksctl delete cluster --name feria --region us-east-1
```

Para un clúster local, `kubectl apply -k deploy/k8s/overlays/local` usa la clase de almacenamiento por defecto.
