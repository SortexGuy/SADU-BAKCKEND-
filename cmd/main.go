package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/internal/handlers"
	"uneg.edu.ve/servicio-sadu-back/internal/middlewares"
	"uneg.edu.ve/servicio-sadu-back/internal/routes"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
)

func main() {
	config.LoadEnv()
	config.ConnectDB()
	config.SyncDB()
	db := config.DB
	 port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	athleteService := services.AthleteService{DB: db}
	athleteHandler := handlers.NewAthleteHandler(&athleteService)

	universityService := services.UniversityServices{DB: db}
	universityHandler := handlers.NewUniversityHandler(&universityService)

	disciplineService := services.DisciplineServices{DB: db}
	disciplineHandler := handlers.NewDisciplineHandler(&disciplineService)

	majorService := services.MajorServices{DB: db}
	majorHandler := handlers.NewMajorHandler(&majorService)

	tourneyService := services.TourneyServices{DB: db}
	tourneyHandler := handlers.NewTourneyHandler(&tourneyService)

	teacherService := services.TeacherService{DB: db}
	teacherHandler := handlers.NewTeacherHandler(&teacherService)

	teamService := services.TeamServices{DB: db}
	teamHandler := handlers.NewTeamHandler(&teamService)

	eventService := services.EventService{DB: db}
	eventHandlers := handlers.NewEventHandler(&eventService)

	userService := services.UserService{DB: db}
	if err := userService.EnsureAdminUser(); err != nil {
		log.Printf("Error ensuring admin user: %v", err)
	}
	userHandlers := handlers.NewUserHandler(&userService)

	r := gin.Default()
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	//configuracion de CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://dominio.uneg.edu.ve"}, // "https://dominio.uneg.edu.ve" es cuando tengamos algun dominio ya puesto
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	/*rutas*/
	routes.RegisterAthletesRoutes(r.Group("/athletes", middlewares.AuthMiddleware()), athleteHandler)
	routes.RegisterUniversityRoutes(r.Group("/universities", middlewares.AuthMiddleware()), universityHandler)
	routes.RegisterDisciplines(r.Group("/disciplines", middlewares.AuthMiddleware()), disciplineHandler)
	routes.RegisterMajorsRoutes(r.Group("/majors", middlewares.AuthMiddleware()), majorHandler)
	routes.RegisterTourney(r.Group("/tourneys", middlewares.AuthMiddleware()), tourneyHandler)
	routes.RegisterTeacherRoutes(r.Group("/teachers", middlewares.AuthMiddleware()), teacherHandler)
	routes.RegisterTeamRoutes(r.Group("/teams", middlewares.AuthMiddleware()), teamHandler)
	routes.RegisterEventsRouters(r.Group("/events"), eventHandlers)
	routes.RegisterUserRoutes(r.Group("/users"), userHandlers)
	log.Println(" Server corriendo en http://localhost:" + port)
	r.Run(":" + port)
	println("Exitted")
}


