package services_test

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

var (
	inicioAgosto = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	finAgosto    = time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
)

// torneoValido es el torneo minimo que acepta el servicio: con disciplina, que es
// obligatoria, y con un rango de fechas coherente.
func torneoValido(cat testutil.Catalogo) schema.Tourney {
	return schema.Tourney{
		Name:         "Copa Rector",
		Status:       schema.StatusWait,
		StartDate:    inicioAgosto,
		EndDate:      finAgosto,
		DisciplineID: cat.Disciplina1ID,
	}
}

// partidos crea eventos de la disciplina del catalogo y devuelve la lista de
// referencias que espera el servicio para asociarlos a un torneo: solo el ID, que
// es lo que arma el handler a partir de los identificadores del cliente.
func partidos(t *testing.T, db *gorm.DB, cat testutil.Catalogo, cuantos int) []schema.Event {
	t.Helper()

	referencias := []schema.Event{}
	for i := 0; i < cuantos; i++ {
		evento := schema.Event{
			Name:                 "Partido",
			Date:                 inicioAgosto.AddDate(0, 0, i),
			Status:               "Pendiente",
			HomeTeamID:           cat.EquipoLocalID,
			OppositeTeamID:       cat.EquipoVisitID,
			ResponsableTeacherID: cat.ProfesorID,
			DisciplineID:         cat.Disciplina1ID,
		}
		testutil.SinError(t, db.Omit("TourneyID").Create(&evento).Error, "crear un partido")
		referencias = append(referencias, schema.Event{Model: gorm.Model{ID: evento.ID}})
	}
	return referencias
}

// torneoDeLosPartidos devuelve el torneo con el que quedo cada partido, en NULL
// como 0, para poder afirmar sobre la asociacion desde el lado del evento.
func torneoDeLosPartidos(t *testing.T, db *gorm.DB) []uint {
	t.Helper()

	var ids []uint
	testutil.SinError(t,
		db.Raw("SELECT COALESCE(tourney_id, 0) FROM events ORDER BY id").Scan(&ids).Error,
		"consultar el torneo de los partidos")
	return ids
}

func TestCrearTorneo(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "contrato de la creacion: disciplina obligatoria, rango de fechas y asociacion de los partidos")

	t.Run("guarda el torneo y le asocia sus partidos", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.TourneyServices{DB: db}

		nuevo := torneoValido(cat)
		nuevo.Events = partidos(t, db, cat, 2)

		creado, err := servicio.CreateTourney(nuevo)
		testutil.SinError(t, err, "crear el torneo")
		testutil.Igual(t, creado.Name, "Copa Rector", "nombre del torneo")
		testutil.Igual(t, creado.DisciplineID, cat.Disciplina1ID, "disciplina del torneo")
		testutil.Igual(t, len(creado.Events), 2, "partidos asociados")

		// La asociacion tiene que verse desde el evento, que es donde vive la clave.
		for _, torneoID := range torneoDeLosPartidos(t, db) {
			testutil.Igual(t, torneoID, creado.ID, "torneo del partido")
		}
	})

	t.Run("sin partidos se guarda igual", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.TourneyServices{DB: db}

		creado, err := servicio.CreateTourney(torneoValido(cat))
		testutil.SinError(t, err, "crear el torneo sin partidos")
		testutil.Igual(t, len(creado.Events), 0, "partidos asociados")
	})

	t.Run("sin disciplina se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		testutil.SilenciarLogs(t)
		servicio := services.TourneyServices{DB: db}

		nuevo := torneoValido(cat)
		nuevo.DisciplineID = 0

		_, err := servicio.CreateTourney(nuevo)
		if !errors.Is(err, services.ErrMissingDiscipline) {
			t.Fatalf("se esperaba ErrMissingDiscipline, se obtuvo: %v", err)
		}

		var guardados int64
		db.Model(&schema.Tourney{}).Count(&guardados)
		testutil.Igual(t, guardados, int64(1), "torneos en la base (solo el del catalogo)")
	})

	t.Run("con una disciplina inexistente se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		testutil.SilenciarLogs(t)
		servicio := services.TourneyServices{DB: db}

		nuevo := torneoValido(cat)
		nuevo.DisciplineID = 999

		_, err := servicio.CreateTourney(nuevo)
		if err == nil {
			t.Fatal("se esperaba error con una disciplina inexistente")
		}
		if !services.IsInvalidReference(err) {
			t.Fatalf("el error deberia reconocerse como referencia invalida: %v", err)
		}
	})
}

func TestRangoDeFechasDelTorneo(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "particiones y valores limite del rango: fin anterior, fin igual, fin posterior y fechas sin definir")

	casos := []struct {
		nombre string
		inicio time.Time
		fin    time.Time
		valido bool
	}{
		{"fin posterior al inicio", inicioAgosto, finAgosto, true},
		{"fin igual al inicio", inicioAgosto, inicioAgosto, true},
		{"fin un dia antes del inicio", inicioAgosto, inicioAgosto.AddDate(0, 0, -1), false},
		{"sin fechas", time.Time{}, time.Time{}, true},
		{"solo inicio", inicioAgosto, time.Time{}, true},
		{"solo fin", time.Time{}, finAgosto, true},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			db := testutil.NuevaDB(t)
			cat := testutil.SembrarCatalogo(t, db)
			servicio := services.TourneyServices{DB: db}

			nuevo := torneoValido(cat)
			nuevo.StartDate, nuevo.EndDate = caso.inicio, caso.fin

			_, err := servicio.CreateTourney(nuevo)
			if caso.valido {
				testutil.SinError(t, err, "crear el torneo")
				return
			}
			testutil.SilenciarLogs(t)
			if !errors.Is(err, services.ErrInvalidDateRange) {
				t.Fatalf("se esperaba ErrInvalidDateRange, se obtuvo: %v", err)
			}
		})
	}
}

func TestEditarTorneo(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "recorre las tres ramas de la lista de partidos: con partidos, vacia y ausente")

	preparar := func(t *testing.T) (*gorm.DB, testutil.Catalogo, schema.Tourney) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.TourneyServices{DB: db}

		nuevo := torneoValido(cat)
		nuevo.Events = partidos(t, db, cat, 2)
		creado, err := servicio.CreateTourney(nuevo)
		testutil.SinError(t, err, "crear el torneo de partida")

		return db, cat, creado
	}

	t.Run("reemplaza la lista de partidos", func(t *testing.T) {
		db, cat, creado := preparar(t)
		servicio := services.TourneyServices{DB: db}

		// Un tercer partido pasa a ser el unico del torneo.
		nuevos := partidos(t, db, cat, 1)
		cambios := schema.Tourney{Name: "Copa Rector 2026", Events: nuevos}

		editado, err := servicio.UpdateTourney(cambios, true, contextoConID(idComoTexto(creado.ID)))
		testutil.SinError(t, err, "editar el torneo")
		testutil.Igual(t, editado.Name, "Copa Rector 2026", "nombre actualizado")
		testutil.Igual(t, len(editado.Events), 1, "partidos tras el reemplazo")
		testutil.Igual(t, editado.Events[0].ID, nuevos[0].ID, "partido que queda")
	})

	t.Run("con la lista vacia el torneo se queda sin partidos", func(t *testing.T) {
		db, _, creado := preparar(t)
		servicio := services.TourneyServices{DB: db}

		cambios := schema.Tourney{Events: []schema.Event{}}

		editado, err := servicio.UpdateTourney(cambios, true, contextoConID(idComoTexto(creado.ID)))
		testutil.SinError(t, err, "quitar todos los partidos")
		testutil.Igual(t, len(editado.Events), 0, "partidos tras vaciar la lista")

		// Los partidos siguen existiendo, solo dejan de pertenecer al torneo.
		var existentes int64
		db.Model(&schema.Event{}).Count(&existentes)
		testutil.Igual(t, existentes, int64(2), "partidos que siguen existiendo")
		for _, torneoID := range torneoDeLosPartidos(t, db) {
			testutil.Igual(t, torneoID, uint(0), "torneo del partido desasociado")
		}
	})

	t.Run("sin mencionar los partidos los conserva", func(t *testing.T) {
		db, _, creado := preparar(t)
		servicio := services.TourneyServices{DB: db}

		cambios := schema.Tourney{Status: schema.StatusON}

		editado, err := servicio.UpdateTourney(cambios, false, contextoConID(idComoTexto(creado.ID)))
		testutil.SinError(t, err, "editar solo el estado")
		testutil.Igual(t, editado.Status, schema.StatusON, "estado actualizado")
		testutil.Igual(t, len(editado.Events), 2, "partidos conservados")
	})

	t.Run("un rango invertido se rechaza aunque solo venga una fecha", func(t *testing.T) {
		// El fin llega solo en la peticion y el inicio ya estaba guardado: la
		// validacion mira los valores que quedarian, no los que trae el cuerpo.
		db, _, creado := preparar(t)
		testutil.SilenciarLogs(t)
		servicio := services.TourneyServices{DB: db}

		cambios := schema.Tourney{EndDate: inicioAgosto.AddDate(0, 0, -1)}

		_, err := servicio.UpdateTourney(cambios, false, contextoConID(idComoTexto(creado.ID)))
		if !errors.Is(err, services.ErrInvalidDateRange) {
			t.Fatalf("se esperaba ErrInvalidDateRange, se obtuvo: %v", err)
		}
	})

	t.Run("un torneo inexistente da error", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.TourneyServices{DB: db}

		_, err := servicio.UpdateTourney(schema.Tourney{Name: "Fantasma"}, false, contextoConID("999"))
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("se esperaba ErrRecordNotFound, se obtuvo: %v", err)
		}
	})
}

func TestDetalleDelTorneo(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "el detalle lleva la disciplina y los partidos que necesita el formulario de edicion")

	db := testutil.NuevaDB(t)
	cat := testutil.SembrarCatalogo(t, db)
	servicio := services.TourneyServices{DB: db}

	nuevo := torneoValido(cat)
	nuevo.Events = partidos(t, db, cat, 2)
	creado, err := servicio.CreateTourney(nuevo)
	testutil.SinError(t, err, "crear el torneo")

	detalle, err := servicio.GetTourneyByID(contextoConID(idComoTexto(creado.ID)))
	testutil.SinError(t, err, "leer el detalle del torneo")
	testutil.Igual(t, detalle.Name, "Copa Rector", "nombre")
	testutil.Igual(t, detalle.DisciplineID, cat.Disciplina1ID, "disciplina")
	testutil.Igual(t, detalle.DisciplineName, "Voleibol", "nombre de la disciplina")
	testutil.Igual(t, len(detalle.Events), 2, "partidos en el detalle")
	testutil.Igual(t, detalle.TotalEvents, uint(2), "total de partidos")
	testutil.Igual(t, detalle.StartDate.Equal(inicioAgosto), true, "fecha de inicio")
	testutil.Igual(t, detalle.EndDate.Equal(finAgosto), true, "fecha de fin")
}

func TestFiltrosDeTorneos(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "recorre cada filtro del listado por separado y combinados")

	preparar := func(t *testing.T) (*gorm.DB, testutil.Catalogo) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.TourneyServices{DB: db}

		voleibol := torneoValido(cat)
		voleibol.Name = "Copa de Voleibol"
		_, err := servicio.CreateTourney(voleibol)
		testutil.SinError(t, err, "crear el torneo de voleibol")

		futbol := torneoValido(cat)
		futbol.Name = "Liga de Futbol"
		futbol.Status = schema.StatusOFF
		futbol.DisciplineID = cat.Disciplina2ID
		_, err = servicio.CreateTourney(futbol)
		testutil.SinError(t, err, "crear el torneo de futbol")

		return db, cat
	}

	// El catalogo ya trae un torneo ("Copa UNEG", Activo, Voleibol), asi que el
	// total sin filtros es tres.
	t.Run("sin filtros devuelve todos", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.TourneyServices{DB: db}

		lista, err := servicio.GetAllTourney("", "", "")
		testutil.SinError(t, err, "listar torneos")
		testutil.Igual(t, len(lista), 3, "torneos")
	})

	t.Run("por nombre", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.TourneyServices{DB: db}

		lista, err := servicio.GetAllTourney("Liga", "", "")
		testutil.SinError(t, err, "listar por nombre")
		testutil.Igual(t, len(lista), 1, "torneos por nombre")
		testutil.Igual(t, lista[0].Name, "Liga de Futbol", "torneo encontrado")
	})

	t.Run("por estado", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.TourneyServices{DB: db}

		lista, err := servicio.GetAllTourney("", string(schema.StatusOFF), "")
		testutil.SinError(t, err, "listar por estado")
		testutil.Igual(t, len(lista), 1, "torneos finalizados")
	})

	t.Run("por disciplina", func(t *testing.T) {
		// Es el filtro de la pagina de torneos: mira tourneys.discipline_id, asi que
		// solo encuentra los torneos que tienen disciplina.
		db, cat := preparar(t)
		servicio := services.TourneyServices{DB: db}

		lista, err := servicio.GetAllTourney("", "", idComoTexto(uint(cat.Disciplina2ID)))
		testutil.SinError(t, err, "listar por disciplina")
		testutil.Igual(t, len(lista), 1, "torneos de la segunda disciplina")
		testutil.Igual(t, lista[0].DisciplineName, "Futbol", "disciplina del torneo")
	})

	t.Run("por nombre y disciplina a la vez", func(t *testing.T) {
		db, cat := preparar(t)
		servicio := services.TourneyServices{DB: db}

		lista, err := servicio.GetAllTourney("Copa", "", idComoTexto(uint(cat.Disciplina1ID)))
		testutil.SinError(t, err, "listar por nombre y disciplina")
		testutil.Igual(t, len(lista), 2, "torneos de voleibol con Copa en el nombre")

		// Los filtros se acumulan con AND: un nombre de voleibol con la disciplina de
		// futbol no devuelve nada.
		ninguno, err := servicio.GetAllTourney("Copa", "", idComoTexto(uint(cat.Disciplina2ID)))
		testutil.SinError(t, err, "combinar nombre y otra disciplina")
		testutil.Igual(t, len(ninguno), 0, "torneos de futbol con Copa en el nombre")
	})

	t.Run("el listado cuenta los partidos", func(t *testing.T) {
		db, cat := preparar(t)
		servicio := services.TourneyServices{DB: db}

		conPartidos := torneoValido(cat)
		conPartidos.Name = "Torneo con partidos"
		conPartidos.Events = partidos(t, db, cat, 3)
		_, err := servicio.CreateTourney(conPartidos)
		testutil.SinError(t, err, "crear el torneo con partidos")

		lista, err := servicio.GetAllTourney("Torneo con partidos", "", "")
		testutil.SinError(t, err, "listar el torneo con partidos")
		testutil.Igual(t, len(lista), 1, "torneos encontrados")
		testutil.Igual(t, lista[0].TotalEvents, uint(3), "partidos contados")
	})
}

func TestEliminarTorneo(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "existe o no existe, y el resultado esperado en cada caso")

	db := testutil.NuevaDB(t)
	cat := testutil.SembrarCatalogo(t, db)
	testutil.SilenciarLogs(t)
	servicio := services.TourneyServices{DB: db}

	creado, err := servicio.CreateTourney(torneoValido(cat))
	testutil.SinError(t, err, "crear el torneo")

	t.Run("elimina el torneo existente", func(t *testing.T) {
		testutil.SinError(t, servicio.DeleteTourney(contextoConID(idComoTexto(creado.ID))), "eliminar el torneo")

		var visibles int64
		db.Model(&schema.Tourney{}).Where("id = ?", creado.ID).Count(&visibles)
		testutil.Igual(t, visibles, int64(0), "torneos visibles tras eliminar")
	})

	t.Run("un torneo inexistente da error", func(t *testing.T) {
		if err := servicio.DeleteTourney(contextoConID("999")); err == nil {
			t.Fatal("eliminar un torneo inexistente deberia dar error")
		}
	})
}
