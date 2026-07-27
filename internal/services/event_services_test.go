package services_test

import (
	"testing"
	"time"

	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

// torneo devuelve el puntero que espera el DTO. El campo es un puntero para
// distinguir "la peticion no menciona el torneo" (nil) de "sin torneo" (0).
func torneo(id schema.RegularIDs) *schema.RegularIDs {
	return &id
}

// sinTorneoEnLaBase consulta si la columna quedo en NULL. Un 0 no referenciaria
// ningun torneo y la clave foranea lo rechazaria.
func sinTorneoEnLaBase(t *testing.T, db *gorm.DB, eventoID uint) bool {
	t.Helper()

	var nulo bool
	testutil.SinError(t,
		db.Raw("SELECT tourney_id IS NULL FROM events WHERE id = ?", eventoID).Scan(&nulo).Error,
		"consultar tourney_id")
	return nulo
}

func eventoValido(cat testutil.Catalogo) schema.EventPOSTandPUTDTO {
	return schema.EventPOSTandPUTDTO{
		Name:                 "Alfa vs Beta",
		Date:                 time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
		Status:               "Pendiente",
		Ubication:            "Gimnasio UNEG",
		HomeTeamID:           cat.EquipoLocalID,
		OppositeTeamID:       cat.EquipoVisitID,
		ResponsableTeacherID: cat.ProfesorID,
		DisciplineID:         cat.Disciplina1ID,
	}
}

func TestCrearEvento(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "comprueba que la columna del torneo queda en NULL y no en 0")

	t.Run("con torneo queda vinculado", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		dto := eventoValido(cat)
		dto.TourneyID = torneo(cat.TorneoID)

		creado, err := servicio.CreateEvent(dto)
		testutil.SinError(t, err, "crear evento con torneo")
		testutil.Igual(t, creado.TourneyID, cat.TorneoID, "torneo del evento")
	})

	t.Run("sin torneo se guarda con la columna vacia", func(t *testing.T) {
		// El torneo es opcional. Un 0 no referenciaria ningun torneo y la clave
		// foranea lo rechazaria, asi que la columna debe quedar en NULL.
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		creado, err := servicio.CreateEvent(eventoValido(cat)) // sin TourneyID
		testutil.SinError(t, err, "crear evento sin torneo")

		if !sinTorneoEnLaBase(t, db, creado.ID) {
			t.Error("el evento sin torneo deberia guardar tourney_id en NULL, no en 0")
		}
	})

	t.Run("con torneo en cero se guarda con la columna vacia", func(t *testing.T) {
		// Es lo que envia el formulario cuando se elige "Sin torneo".
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		dto := eventoValido(cat)
		dto.TourneyID = torneo(0)

		creado, err := servicio.CreateEvent(dto)
		testutil.SinError(t, err, "crear evento con torneo en cero")

		if !sinTorneoEnLaBase(t, db, creado.ID) {
			t.Error("un torneo en 0 deberia guardarse como NULL")
		}
	})

	t.Run("con torneo inexistente se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		testutil.SilenciarLogs(t)
		servicio := services.EventService{DB: db}

		dto := eventoValido(cat)
		dto.TourneyID = torneo(999)

		_, err := servicio.CreateEvent(dto)
		if err == nil {
			t.Fatal("se esperaba error con un torneo inexistente")
		}
		if !services.IsInvalidReference(err) {
			t.Fatalf("el error deberia reconocerse como referencia invalida: %v", err)
		}
	})

	t.Run("con equipo inexistente se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		testutil.SilenciarLogs(t)
		servicio := services.EventService{DB: db}

		dto := eventoValido(cat)
		dto.HomeTeamID = 999

		if _, err := servicio.CreateEvent(dto); err == nil {
			t.Fatal("se esperaba error con un equipo inexistente")
		}
	})
}

func TestEditarEvento(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "recorre las tres ramas del torneo al editar: ausente, en cero y con valor")

	t.Run("actualiza los campos y conserva el torneo vacio", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		creado, err := servicio.CreateEvent(eventoValido(cat))
		testutil.SinError(t, err, "crear evento")

		cambios := eventoValido(cat)
		cambios.Name = "Alfa vs Beta (reprogramado)"
		cambios.Ubication = "Cancha 2"

		editado, err := servicio.EditEvent(contextoConID(idComoTexto(creado.ID)), cambios)
		testutil.SinError(t, err, "editar evento")
		testutil.Igual(t, editado.Name, "Alfa vs Beta (reprogramado)", "nombre actualizado")
		testutil.Igual(t, editado.Ubication, "Cancha 2", "sede actualizada")

		if !sinTorneoEnLaBase(t, db, creado.ID) {
			t.Error("al editar sin torneo la columna deberia seguir en NULL")
		}
	})

	t.Run("una peticion que no menciona el torneo lo conserva", func(t *testing.T) {
		// Sin esto, editar el marcador de un partido lo sacaba de su torneo: el DTO
		// llegaba con el torneo en cero y la columna se ponia en NULL.
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		inicial := eventoValido(cat)
		inicial.TourneyID = torneo(cat.TorneoID)
		creado, err := servicio.CreateEvent(inicial)
		testutil.SinError(t, err, "crear evento con torneo")

		cambios := eventoValido(cat) // TourneyID nil: la peticion no habla del torneo
		cambios.HomePoints = 3
		cambios.OppositePoints = 1

		editado, err := servicio.EditEvent(contextoConID(idComoTexto(creado.ID)), cambios)
		testutil.SinError(t, err, "editar el marcador")
		testutil.Igual(t, editado.HomePoints, uint8(3), "puntos del local")
		testutil.Igual(t, editado.TourneyID, cat.TorneoID, "torneo tras editar el marcador")
	})

	t.Run("con el torneo en cero lo desvincula", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		inicial := eventoValido(cat)
		inicial.TourneyID = torneo(cat.TorneoID)
		creado, err := servicio.CreateEvent(inicial)
		testutil.SinError(t, err, "crear evento con torneo")

		cambios := eventoValido(cat)
		cambios.TourneyID = torneo(0) // "Sin torneo" en el formulario

		_, err = servicio.EditEvent(contextoConID(idComoTexto(creado.ID)), cambios)
		testutil.SinError(t, err, "quitar el torneo")

		if !sinTorneoEnLaBase(t, db, creado.ID) {
			t.Error("quitar el torneo deberia dejar tourney_id en NULL")
		}
	})

	t.Run("con otro torneo lo cambia", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		otro := schema.Tourney{
			Name: "Copa Rector", Status: schema.StatusWait,
			DisciplineID: cat.Disciplina1ID,
		}
		testutil.SinError(t, db.Create(&otro).Error, "crear el segundo torneo")

		inicial := eventoValido(cat)
		inicial.TourneyID = torneo(cat.TorneoID)
		creado, err := servicio.CreateEvent(inicial)
		testutil.SinError(t, err, "crear evento con torneo")

		cambios := eventoValido(cat)
		cambios.TourneyID = torneo(schema.RegularIDs(otro.ID))

		editado, err := servicio.EditEvent(contextoConID(idComoTexto(creado.ID)), cambios)
		testutil.SinError(t, err, "cambiar el torneo")
		testutil.Igual(t, editado.TourneyID, schema.RegularIDs(otro.ID), "torneo nuevo")
	})

	t.Run("un evento inexistente da error", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		testutil.SilenciarLogs(t)
		servicio := services.EventService{DB: db}

		if _, err := servicio.EditEvent(contextoConID("999"), eventoValido(cat)); err == nil {
			t.Fatal("se esperaba error al editar un evento inexistente")
		}
	})
}

func TestEliminarEvento(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "existe o no existe, y el resultado esperado en cada caso")

	db := testutil.NuevaDB(t)
	cat := testutil.SembrarCatalogo(t, db)
	testutil.SilenciarLogs(t)
	servicio := services.EventService{DB: db}

	creado, err := servicio.CreateEvent(eventoValido(cat))
	testutil.SinError(t, err, "crear evento")

	t.Run("elimina el evento existente", func(t *testing.T) {
		testutil.SinError(t, servicio.DeleteEvent(contextoConID(idComoTexto(creado.ID))), "eliminar evento")

		var visibles int64
		db.Model(&schema.Event{}).Count(&visibles)
		testutil.Igual(t, visibles, int64(0), "eventos visibles tras eliminar")
	})

	t.Run("un evento inexistente da error", func(t *testing.T) {
		if err := servicio.DeleteEvent(contextoConID("999")); err == nil {
			t.Fatal("eliminar un evento inexistente deberia dar error")
		}
	})
}

func TestFiltrosDeEventos(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "recorre cada filtro y la agrupacion del OR entre equipo local y visitante")

	preparar := func(t *testing.T) (*gorm.DB, testutil.Catalogo) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		agosto := eventoValido(cat)
		agosto.Name = "Final de voleibol"
		agosto.Status = "Pendiente"
		agosto.TourneyID = torneo(cat.TorneoID)
		_, err := servicio.CreateEvent(agosto)
		testutil.SinError(t, err, "crear el evento de agosto")

		septiembre := eventoValido(cat)
		septiembre.Name = "Amistoso de futbol"
		septiembre.Status = "Finalizado"
		septiembre.DisciplineID = cat.Disciplina2ID
		septiembre.Date = time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
		_, err = servicio.CreateEvent(septiembre)
		testutil.SinError(t, err, "crear el evento de septiembre")

		return db, cat
	}

	t.Run("sin filtros devuelve todos", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		lista, err := servicio.GetEvents(0, "", "", "", "", "", "", "")
		testutil.SinError(t, err, "listar eventos")
		testutil.Igual(t, len(lista), 2, "eventos")
	})

	t.Run("por identificador devuelve solo uno", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		var primero schema.Event
		testutil.SinError(t, db.First(&primero).Error, "leer el primer evento")

		lista, err := servicio.GetEvents(primero.ID, "", "", "", "", "", "", "")
		testutil.SinError(t, err, "listar por id")
		testutil.Igual(t, len(lista), 1, "eventos por id")
	})

	t.Run("por nombre", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		lista, err := servicio.GetEvents(0, "voleibol", "", "", "", "", "", "")
		testutil.SinError(t, err, "filtrar por nombre")
		testutil.Igual(t, len(lista), 1, "eventos con 'voleibol' en el nombre")
	})

	t.Run("por estado", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		lista, err := servicio.GetEvents(0, "", "Finalizado", "", "", "", "", "")
		testutil.SinError(t, err, "filtrar por estado")
		testutil.Igual(t, len(lista), 1, "eventos finalizados")
	})

	t.Run("por disciplina", func(t *testing.T) {
		db, cat := preparar(t)
		servicio := services.EventService{DB: db}

		lista, err := servicio.GetEvents(0, "", "", idComoTexto(uint(cat.Disciplina2ID)), "", "", "", "")
		testutil.SinError(t, err, "filtrar por disciplina")
		testutil.Igual(t, len(lista), 1, "eventos de la disciplina 2")
	})

	t.Run("por rango de fechas", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		agosto, err := servicio.GetEvents(0, "", "", "", "2026-08-01", "2026-08-31", "", "")
		testutil.SinError(t, err, "filtrar por agosto")
		testutil.Igual(t, len(agosto), 1, "eventos de agosto")

		desdeSeptiembre, err := servicio.GetEvents(0, "", "", "", "2026-09-01", "", "", "")
		testutil.SinError(t, err, "filtrar desde septiembre")
		testutil.Igual(t, len(desdeSeptiembre), 1, "eventos desde septiembre")

		futuro, err := servicio.GetEvents(0, "", "", "", "2027-01-01", "", "", "")
		testutil.SinError(t, err, "filtrar desde 2027")
		testutil.Igual(t, len(futuro), 0, "eventos de 2027")
	})

	t.Run("por nombre de equipo, que busca en local y visitante", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		local, err := servicio.GetEvents(0, "", "", "", "", "", "Alfa", "")
		testutil.SinError(t, err, "filtrar por equipo local")
		testutil.Igual(t, len(local), 2, "eventos del equipo local")

		visitante, err := servicio.GetEvents(0, "", "", "", "", "", "Beta", "")
		testutil.SinError(t, err, "filtrar por equipo visitante")
		testutil.Igual(t, len(visitante), 2, "eventos del equipo visitante")

		inexistente, err := servicio.GetEvents(0, "", "", "", "", "", "Gamma", "")
		testutil.SinError(t, err, "filtrar por un equipo que no juega")
		testutil.Igual(t, len(inexistente), 0, "eventos de un equipo inexistente")
	})

	t.Run("por universidad", func(t *testing.T) {
		db, cat := preparar(t)
		servicio := services.EventService{DB: db}

		propios, err := servicio.GetEvents(0, "", "", "", "", "", "", idComoTexto(uint(cat.UniversidadID)))
		testutil.SinError(t, err, "filtrar por universidad")
		testutil.Igual(t, len(propios), 2, "eventos de la universidad")

		otra, err := servicio.GetEvents(0, "", "", "", "", "", "", "999")
		testutil.SinError(t, err, "filtrar por otra universidad")
		testutil.Igual(t, len(otra), 0, "eventos de una universidad sin equipos")
	})

	t.Run("el OR del filtro de equipo no se mezcla con los demas", func(t *testing.T) {
		// team_name usa OR entre local y visitante. Si ese OR no estuviera entre
		// parentesis, se mezclaria con el AND del estado y devolveria de mas.
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		lista, err := servicio.GetEvents(0, "", "Finalizado", "", "", "", "Alfa", "")
		testutil.SinError(t, err, "combinar estado y equipo")
		testutil.Igual(t, len(lista), 1, "eventos finalizados del equipo Alfa")
	})

	t.Run("el listado precarga los equipos y la disciplina", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.EventService{DB: db}

		lista, err := servicio.GetEvents(0, "voleibol", "", "", "", "", "", "")
		testutil.SinError(t, err, "listar eventos")
		if len(lista) != 1 {
			t.Fatalf("se esperaba un evento y se obtuvieron %d", len(lista))
		}
		testutil.Igual(t, lista[0].HomeTeam.Name, "UNEG Alfa", "equipo local precargado")
		testutil.Igual(t, lista[0].Discipline.Name, "Voleibol", "disciplina precargada")
		testutil.Igual(t, lista[0].Tourney.Name, "Copa UNEG", "torneo precargado")
	})
}

// fechasDelTorneo lee el rango guardado, que es lo que el ajuste automatico
// modifica por detras del alta o la edicion del partido.
func fechasDelTorneo(t *testing.T, db *gorm.DB, torneoID schema.RegularIDs) (time.Time, time.Time) {
	t.Helper()

	var guardado schema.Tourney
	testutil.SinError(t, db.First(&guardado, torneoID).Error, "leer el torneo")
	return guardado.StartDate.UTC(), guardado.EndDate.UTC()
}

// dia construye la medianoche UTC con la que se guardan las fechas del torneo.
func dia(anio int, mes time.Month, numero int) time.Time {
	return time.Date(anio, mes, numero, 0, 0, 0, 0, time.UTC)
}

func TestAjusteDeFechasDelTorneo(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "recorre las ramas del ajuste: torneo sin fechas, partido antes del inicio, despues del fin y dentro del rango")

	// conFechas deja el torneo del catalogo con un rango ya definido, que es el caso
	// en el que el ajuste tiene que decidir si lo amplia o lo deja como esta.
	conFechas := func(t *testing.T, db *gorm.DB, torneoID schema.RegularIDs, inicio, fin time.Time) {
		t.Helper()
		testutil.SinError(t,
			db.Model(&schema.Tourney{}).Where("id = ?", torneoID).
				Updates(map[string]interface{}{"start_date": inicio, "end_date": fin}).Error,
			"fijar las fechas del torneo")
	}

	t.Run("un torneo sin fechas toma la del partido en las dos", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db) // el torneo del catalogo no trae fechas
		servicio := services.EventService{DB: db}

		dto := eventoValido(cat) // 15 de agosto de 2026, 10:00
		dto.TourneyID = torneo(cat.TorneoID)
		_, err := servicio.CreateEvent(dto)
		testutil.SinError(t, err, "crear evento con torneo")

		inicio, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, inicio, dia(2026, time.August, 15), "inicio del torneo")
		testutil.Igual(t, fin, dia(2026, time.August, 15), "fin del torneo")
	})

	t.Run("la fecha se guarda truncada al dia", func(t *testing.T) {
		// El formulario de torneos usa <input type="date">, asi que la hora del
		// partido no debe colarse dentro del rango.
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		dto := eventoValido(cat)
		dto.Date = time.Date(2026, 8, 15, 18, 30, 0, 0, time.UTC)
		dto.TourneyID = torneo(cat.TorneoID)
		_, err := servicio.CreateEvent(dto)
		testutil.SinError(t, err, "crear evento con torneo")

		_, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, fin, dia(2026, time.August, 15), "fin del torneo sin la hora del partido")
	})

	t.Run("un partido anterior adelanta el inicio", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}
		conFechas(t, db, cat.TorneoID, dia(2026, time.August, 10), dia(2026, time.August, 20))

		dto := eventoValido(cat)
		dto.Date = time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)
		dto.TourneyID = torneo(cat.TorneoID)
		_, err := servicio.CreateEvent(dto)
		testutil.SinError(t, err, "crear evento anterior al inicio")

		inicio, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, inicio, dia(2026, time.August, 5), "inicio adelantado al partido")
		testutil.Igual(t, fin, dia(2026, time.August, 20), "fin sin tocar")
	})

	t.Run("un partido posterior atrasa el fin", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}
		conFechas(t, db, cat.TorneoID, dia(2026, time.August, 10), dia(2026, time.August, 20))

		dto := eventoValido(cat)
		dto.Date = time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
		dto.TourneyID = torneo(cat.TorneoID)
		_, err := servicio.CreateEvent(dto)
		testutil.SinError(t, err, "crear evento posterior al fin")

		inicio, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, inicio, dia(2026, time.August, 10), "inicio sin tocar")
		testutil.Igual(t, fin, dia(2026, time.August, 25), "fin atrasado al partido")
	})

	t.Run("un partido dentro del rango no lo cambia", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}
		conFechas(t, db, cat.TorneoID, dia(2026, time.August, 10), dia(2026, time.August, 20))

		dto := eventoValido(cat) // 15 de agosto
		dto.TourneyID = torneo(cat.TorneoID)
		_, err := servicio.CreateEvent(dto)
		testutil.SinError(t, err, "crear evento dentro del rango")

		inicio, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, inicio, dia(2026, time.August, 10), "inicio sin tocar")
		testutil.Igual(t, fin, dia(2026, time.August, 20), "fin sin tocar")
	})

	t.Run("mover la fecha del partido estira el torneo", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}
		conFechas(t, db, cat.TorneoID, dia(2026, time.August, 10), dia(2026, time.August, 20))

		inicial := eventoValido(cat)
		inicial.TourneyID = torneo(cat.TorneoID)
		creado, err := servicio.CreateEvent(inicial)
		testutil.SinError(t, err, "crear evento con torneo")

		cambios := eventoValido(cat)
		cambios.TourneyID = torneo(cat.TorneoID)
		cambios.Date = time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)
		_, err = servicio.EditEvent(contextoConID(idComoTexto(creado.ID)), cambios)
		testutil.SinError(t, err, "reprogramar el partido")

		_, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, fin, dia(2026, time.September, 2), "fin estirado al partido reprogramado")
	})

	t.Run("editar sin mencionar el torneo tambien lo ajusta", func(t *testing.T) {
		// El DTO llega con el torneo en nil, pero el partido sigue perteneciendo al
		// mismo torneo y su fecha nueva tiene que caber en el rango.
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}
		conFechas(t, db, cat.TorneoID, dia(2026, time.August, 10), dia(2026, time.August, 20))

		inicial := eventoValido(cat)
		inicial.TourneyID = torneo(cat.TorneoID)
		creado, err := servicio.CreateEvent(inicial)
		testutil.SinError(t, err, "crear evento con torneo")

		cambios := eventoValido(cat) // TourneyID nil
		cambios.Date = time.Date(2026, 8, 1, 15, 0, 0, 0, time.UTC)
		_, err = servicio.EditEvent(contextoConID(idComoTexto(creado.ID)), cambios)
		testutil.SinError(t, err, "reprogramar el partido")

		inicio, _ := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, inicio, dia(2026, time.August, 1), "inicio adelantado al partido")
	})

	t.Run("el torneo del que sale el partido conserva sus fechas", func(t *testing.T) {
		// El rango solo se amplia: encogerlo al desvincular borraria fechas puestas a
		// mano y dejaria fuera a los demas partidos.
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}

		otro := schema.Tourney{
			Name: "Copa Rector", Status: schema.StatusWait,
			DisciplineID: cat.Disciplina1ID,
		}
		testutil.SinError(t, db.Create(&otro).Error, "crear el segundo torneo")

		inicial := eventoValido(cat)
		inicial.TourneyID = torneo(cat.TorneoID)
		creado, err := servicio.CreateEvent(inicial)
		testutil.SinError(t, err, "crear evento con torneo")

		cambios := eventoValido(cat)
		cambios.TourneyID = torneo(schema.RegularIDs(otro.ID))
		_, err = servicio.EditEvent(contextoConID(idComoTexto(creado.ID)), cambios)
		testutil.SinError(t, err, "mover el partido de torneo")

		inicio, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, inicio, dia(2026, time.August, 15), "el torneo original conserva su inicio")
		testutil.Igual(t, fin, dia(2026, time.August, 15), "el torneo original conserva su fin")

		inicioNuevo, finNuevo := fechasDelTorneo(t, db, schema.RegularIDs(otro.ID))
		testutil.Igual(t, inicioNuevo, dia(2026, time.August, 15), "el torneo destino toma la fecha del partido")
		testutil.Igual(t, finNuevo, dia(2026, time.August, 15), "el torneo destino toma la fecha del partido")
	})

	t.Run("un partido sin torneo no ajusta nada", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.EventService{DB: db}
		conFechas(t, db, cat.TorneoID, dia(2026, time.August, 10), dia(2026, time.August, 20))

		dto := eventoValido(cat)
		dto.Date = time.Date(2027, 1, 1, 9, 0, 0, 0, time.UTC)
		_, err := servicio.CreateEvent(dto) // sin torneo
		testutil.SinError(t, err, "crear evento sin torneo")

		inicio, fin := fechasDelTorneo(t, db, cat.TorneoID)
		testutil.Igual(t, inicio, dia(2026, time.August, 10), "inicio sin tocar")
		testutil.Igual(t, fin, dia(2026, time.August, 20), "fin sin tocar")
	})
}
