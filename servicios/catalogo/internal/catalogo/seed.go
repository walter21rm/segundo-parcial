package catalogo

import "context"

type productorSeed struct {
	ID, Nombre, Distrito, Telefono, Descripcion string
}

type productoSeed struct {
	ID, ProductorID, Nombre, Categoria, Unidad, Stock, Descripcion string
	Precio                                                         int
}

func (s *Store) Sembrar(ctx context.Context) error {
	for _, p := range productoresSemilla {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO productores (id, nombre, distrito, telefono, descripcion)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO NOTHING`, p.ID, p.Nombre, p.Distrito, p.Telefono, p.Descripcion); err != nil {
			return err
		}
	}
	for _, p := range productosSemilla {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO productos (id, productor_id, nombre, categoria, unidad, precio_centavos, stock, descripcion)
			VALUES ($1, $2, $3, $4, $5, $6, $7::numeric, $8)
			ON CONFLICT (id) DO NOTHING`,
			p.ID, p.ProductorID, p.Nombre, p.Categoria, p.Unidad, p.Precio, p.Stock, p.Descripcion); err != nil {
			return err
		}
	}
	return nil
}

var productoresSemilla = []productorSeed{
	{"11111111-1111-4111-8111-111111111111", "Huerta Santa Rosa", "Santiago de Surco", "+51941111001", "Hojas y hierbas cortadas de madrugada en Surco."},
	{"22222222-2222-4222-8222-222222222222", "Fundo La Molina", "La Molina", "+51941111002", "Fruta de estación del fundo, sin cámara de frío de por medio."},
	{"33333333-3333-4333-8333-333333333333", "Quesería Barranco", "Barranco", "+51941111003", "Lácteos del día, se acaban antes del mediodía."},
	{"44444444-4444-4444-8444-444444444444", "Apiario Pachacámac", "Pachacámac", "+51941111004", "Miel de colmenas propias, envasada en el apiario."},
	{"55555555-5555-4555-8555-555555555555", "Granos del Valle", "Lurín", "+51941111005", "Quinua y maíz que bajan del valle el mismo día."},
}

var productosSemilla = []productoSeed{
	{"10000000-0000-4000-8000-000000000001", "11111111-1111-4111-8111-111111111111", "Espinaca criolla", "verduras", "atado", "40.000", "Manojo lavado, para saltear o para crema.", 350},
	{"10000000-0000-4000-8000-000000000002", "11111111-1111-4111-8111-111111111111", "Tomate italiano", "verduras", "kg", "25.000", "Maduro, para ensalada o salsa.", 450},
	{"10000000-0000-4000-8000-000000000003", "11111111-1111-4111-8111-111111111111", "Lechuga seda", "verduras", "unidad", "60.000", "Cabeza entera.", 200},
	{"10000000-0000-4000-8000-000000000004", "11111111-1111-4111-8111-111111111111", "Albahaca", "verduras", "atado", "30.000", "Atado chico, aromática.", 300},
	{"10000000-0000-4000-8000-000000000005", "22222222-2222-4222-8222-222222222222", "Palta Fuerte", "frutas", "kg", "18.500", "Cremosa, de la semana.", 890},
	{"10000000-0000-4000-8000-000000000006", "22222222-2222-4222-8222-222222222222", "Fresa", "frutas", "kg", "8.000", "Cosecha de la mañana.", 1200},
	{"10000000-0000-4000-8000-000000000007", "22222222-2222-4222-8222-222222222222", "Plátano de seda", "frutas", "kg", "40.000", "Maduro, para comer al paso.", 320},
	{"10000000-0000-4000-8000-000000000008", "22222222-2222-4222-8222-222222222222", "Limón sutil", "frutas", "kg", "30.000", "De cáscara delgada.", 280},
	{"10000000-0000-4000-8000-000000000009", "33333333-3333-4333-8333-333333333333", "Queso fresco", "lacteos", "kg", "6.000", "Sin sal extra. Se vende al peso.", 2800},
	{"10000000-0000-4000-8000-00000000000a", "33333333-3333-4333-8333-333333333333", "Yogur natural", "lacteos", "litro", "20.000", "Sin azúcar, en botella de litro.", 750},
	{"10000000-0000-4000-8000-00000000000b", "44444444-4444-4444-8444-444444444444", "Miel de eucalipto", "miel", "litro", "12.000", "Clara y fluida.", 3500},
	{"10000000-0000-4000-8000-00000000000c", "44444444-4444-4444-8444-444444444444", "Miel de algarrobo", "miel", "litro", "4.500", "Oscura, de la costa.", 4200},
	{"10000000-0000-4000-8000-00000000000d", "55555555-5555-4555-8555-555555555555", "Quinua blanca", "granos", "kg", "22.000", "Lavada, lista para la olla.", 1500},
	{"10000000-0000-4000-8000-00000000000e", "55555555-5555-4555-8555-555555555555", "Maíz morado", "granos", "kg", "15.000", "Para chicha o mazamorra.", 380},
}
