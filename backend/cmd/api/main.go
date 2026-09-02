package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/oz-fatma/kontrata/backend/graph"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/config"
	"github.com/oz-fatma/kontrata/backend/internal/mailer"
	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
	"github.com/oz-fatma/kontrata/backend/internal/service"
)

// version derleme sırasında ldflags ile gömülür; verilmezse "dev".
var version = "dev"

func main() {
	// Yerel geliştirmede .env varsa yükle. Dosya yoksa (üretim) sessizce devam.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("ortam dosyası okunamadı: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("yapılandırma yüklenemedi: %v", err)
	}
	log.Printf("yapılandırma yüklendi %s sürüm=%s", cfg.String(), version)

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := mongo.Connect(connectCtx, cfg.MongoURI)
	connectCancel()
	repo := repository.NewContractRepository(db)
	kullanicilar := repository.NewUserRepository(db)
	tokenlar := repository.NewVerificationTokenRepository(db)
	mfaKodlari := repository.NewMFACodeRepository(db)
	oturumlar := repository.NewSessionRepository(db)
	cihazlar := repository.NewDeviceRepository(db)
	organizasyonlar := repository.NewOrganizationRepository(db)
	davetler := repository.NewInviteRepository(db)
	denetim := repository.NewAuditRepository(db)
	if err != nil {
		log.Printf("%v; sunucu degraded başlıyor", err)
		repo = repository.NewContractRepository(nil)
		kullanicilar = repository.NewUserRepository(nil)
		tokenlar = repository.NewVerificationTokenRepository(nil)
		mfaKodlari = repository.NewMFACodeRepository(nil)
		oturumlar = repository.NewSessionRepository(nil)
		cihazlar = repository.NewDeviceRepository(nil)
		organizasyonlar = repository.NewOrganizationRepository(nil)
		davetler = repository.NewInviteRepository(nil)
		denetim = repository.NewAuditRepository(nil)
	} else {
		log.Printf("veritabanına bağlanıldı")
		idxCtx, idxCancel := context.WithTimeout(context.Background(), 10*time.Second)
		for _, ensure := range []func(context.Context) error{
			repo.EnsureIndexes,
			kullanicilar.EnsureIndexes,
			tokenlar.EnsureIndexes,
			mfaKodlari.EnsureIndexes,
			oturumlar.EnsureIndexes,
			cihazlar.EnsureIndexes,
			organizasyonlar.EnsureIndexes,
			davetler.EnsureIndexes,
			denetim.EnsureIndexes,
		} {
			if err := ensure(idxCtx); err != nil {
				log.Printf("indeksler oluşturulamadı: %v", err)
			}
		}
		if err := kullanicilar.BackfillAccountFields(idxCtx); err != nil {
			log.Printf("hesap alanı geçişi başarısız: %v", err)
		}
		if err := backfillSozlesmeOrg(idxCtx, kullanicilar, repo); err != nil {
			log.Printf("sözleşme organizasyon geçişi başarısız: %v", err)
		}
		if err := oturumlar.RevokeMissingDevice(idxCtx); err != nil {
			log.Printf("eski oturum geçişi başarısız: %v", err)
		}
		idxCancel()
	}
	signer, err := auth.NewJWT(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("yapılandırma yüklenemedi: %v", err)
	}
	sozlesmeler := service.NewContractService(repo, kullanicilar)
	authSvc := service.NewAuthService(kullanicilar, tokenlar, mfaKodlari, oturumlar, cihazlar, repo, organizasyonlar, davetler, denetim, mailer.New(cfg.Mailer, cfg.SMTP), cfg.Argon2, signer, db)

	srv := newServer(cfg, db, sozlesmeler, authSvc)

	go func() {
		log.Printf("sunucu dinliyor addr=:%d", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("sunucu durdu: %v", err)
		}
	}()

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)
	sig := <-wait
	log.Printf("kapanış sinyali alındı signal=%s", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("düzgün kapanış başarısız: %v", err)
	}

	discCtx, discCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer discCancel()
	if err := db.Disconnect(discCtx); err != nil {
		log.Printf("veritabanı bağlantısı kapatılamadı: %v", err)
	}
	log.Printf("sunucu durdu")
}

func backfillSozlesmeOrg(ctx context.Context, users *repository.UserRepository, soz *repository.ContractRepository) error {
	uyeler, err := users.ListCorporate(ctx)
	if err != nil {
		return err
	}
	for i := range uyeler {
		if uyeler[i].OrganizationID.IsZero() {
			continue
		}
		if err := soz.BackfillOrganization(ctx, uyeler[i].ID, uyeler[i].OrganizationID); err != nil {
			return err
		}
	}
	return nil
}

func newServer(cfg config.Config, db *mongo.Client, sozlesmeler *service.ContractService, authSvc *service.AuthService) *http.Server {
	r := chi.NewRouter()
	r.Use(recoverPanic)
	r.Use(logRequest)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		handleHealthz(w, r, db)
	})
	graph.RegisterRoutes(r, sozlesmeler, authSvc, cfg.Playground)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

type healthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Database string `json:"database"`
}

func handleHealthz(w http.ResponseWriter, r *http.Request, db *mongo.Client) {
	pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	resp := healthResponse{
		Status:   "ok",
		Version:  version,
		Database: "connected",
	}
	code := http.StatusOK
	if err := db.Ping(pingCtx); err != nil {
		resp.Status = "degraded"
		resp.Database = "unreachable"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("healthz yanıtı yazılamadı: %v", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		// Yalnızca metot, yol, durum ve süre. Gövde, sorgu ve başlık yok.
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			log.Printf("panic kurtarıldı: %v", rec)
			http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}
