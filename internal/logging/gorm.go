package logging

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// UmbralConsultaLenta es el tiempo a partir del cual una consulta se registra
// como lenta, con nivel de advertencia.
const UmbralConsultaLenta = 200 * time.Millisecond

// registroGORM adapta el logger de GORM a slog, para que los errores de base de
// datos y las consultas lentas salgan en el mismo flujo estructurado que el resto
// del servidor, en lugar del formato propio de GORM.
type registroGORM struct {
	nivel gormlogger.LogLevel
}

// NuevoGORM devuelve el logger que hay que pasarle a gorm.Config.
func NuevoGORM() gormlogger.Interface {
	return registroGORM{nivel: gormlogger.Warn}
}

func (r registroGORM) LogMode(nivel gormlogger.LogLevel) gormlogger.Interface {
	r.nivel = nivel
	return r
}

func (r registroGORM) Info(ctx context.Context, msg string, datos ...any) {
	if r.nivel >= gormlogger.Info {
		Desde(ctx).Info("gorm: "+Sanear(msg), "detalle", datos)
	}
}

func (r registroGORM) Warn(ctx context.Context, msg string, datos ...any) {
	if r.nivel >= gormlogger.Warn {
		Desde(ctx).Warn("gorm: "+Sanear(msg), "detalle", datos)
	}
}

func (r registroGORM) Error(ctx context.Context, msg string, datos ...any) {
	if r.nivel >= gormlogger.Error {
		Desde(ctx).Error("gorm: "+Sanear(msg), "detalle", datos)
	}
}

// Trace se invoca al terminar cada consulta. Distingue tres casos: error real,
// consulta lenta y consulta normal, y solo incluye la sentencia SQL en los dos
// primeros o cuando el nivel es depuracion.
func (r registroGORM) Trace(ctx context.Context, inicio time.Time, fc func() (string, int64), err error) {
	transcurrido := time.Since(inicio)
	registro := Desde(ctx)

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, filas := fc()
		registro.Error("consulta con error",
			"error", err.Error(),
			"sql", Sanear(sql),
			"filas", filas,
			"duracion_ms", transcurrido.Milliseconds(),
		)

	case errors.Is(err, gorm.ErrRecordNotFound):
		// No es un fallo: es la respuesta normal de un First() sin resultados.
		registro.Debug("consulta sin resultados", "duracion_ms", transcurrido.Milliseconds())

	case transcurrido > UmbralConsultaLenta:
		sql, filas := fc()
		registro.Warn("consulta lenta",
			"sql", Sanear(sql),
			"filas", filas,
			"duracion_ms", transcurrido.Milliseconds(),
			"umbral_ms", UmbralConsultaLenta.Milliseconds(),
		)

	default:
		if registro.Enabled(ctx, slog.LevelDebug) {
			sql, filas := fc()
			registro.Debug("consulta",
				"sql", Sanear(sql),
				"filas", filas,
				"duracion_ms", transcurrido.Milliseconds(),
			)
		}
	}
}
