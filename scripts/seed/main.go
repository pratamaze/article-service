package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

const (
	giantID       = "author-giant"
	giantShare    = 0.30 // 30% artikel milik satu author
	tailAvg       = 30   // rata-rata artikel per author ekor
	spanDays      = 250  // rentang waktu data
	dupTimeShare  = 0.05 // 5% baris punya created_at sama dengan baris sebelumnya
	tailWordShare = 0.35 // porsi kata sintetis (kosakata ekor)
	tailVocabSize = 20000
)

// Kata-kata ini TIDAK memuat kata uji (berita, kebijakan, anggaran, hidrokarbon).
var idHead = strings.Fields(`yang dan di dengan untuk dari pada ini itu adalah akan juga tidak oleh
sebagai dalam karena telah ke mereka kami pemerintah warga kota jakarta harga pasar program baru
lokal ekonomi pendidikan kesehatan jalan hujan banjir macet listrik desa sekolah rumah tahun bulan
minggu hari meluncurkan startup`)

var enHead = strings.Fields(`the and of to in is that for with on as are was by from at this will has
have government city market price program new local economy education health road rain flood traffic
power village school house year month week day report said launched startup`)

type gen struct {
	r               *rand.Rand
	ent             *ulid.MonotonicEntropy
	zID, zEN, zTail *rand.Zipf
	tail            []string
}

func newGen(seed int64) *gen {
	r := rand.New(rand.NewSource(seed))
	g := &gen{r: r, ent: ulid.Monotonic(r, 0)}
	g.zID = rand.NewZipf(r, 1.2, 1, uint64(len(idHead)-1))
	g.zEN = rand.NewZipf(r, 1.2, 1, uint64(len(enHead)-1))
	g.zTail = rand.NewZipf(r, 1.1, 1, tailVocabSize-1)
	g.tail = make([]string, tailVocabSize)
	for i := range g.tail {
		g.tail[i] = fmt.Sprintf("tk%05d", i)
	}
	return g
}

// lang: 0 = Indonesia, 1 = Inggris, 2 = campur
func (g *gen) word(lang int) string {
	if g.r.Float64() < tailWordShare {
		return g.tail[g.zTail.Uint64()]
	}
	useID := lang == 0 || (lang == 2 && g.r.Intn(2) == 0)
	if useID {
		return idHead[g.zID.Uint64()]
	}
	return enHead[g.zEN.Uint64()]
}

func (g *gen) text(lo, hi, lang int) string {
	n := lo + g.r.Intn(hi-lo+1)
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(g.word(lang))
	}
	return b.String()
}

func pickLang(r *rand.Rand) int {
	switch x := r.Float64(); {
	case x < 0.4:
		return 0
	case x < 0.8:
		return 1
	default:
		return 2
	}
}

func buildAuthors(n int) (rows [][]any, tailIDs []string) {
	m := int(float64(n) * (1 - giantShare) / tailAvg)
	if m < 60 {
		m = 60 // supaya author khusus di bawah muat
	}
	rows = append(rows, []any{giantID, "Syndication"})
	budi := []string{"Budi Santoso", " budi   santoso", "BUDI SANTOSO"}
	for i := 1; i <= m; i++ {
		id := fmt.Sprintf("author-%06d", i)
		var name string
		switch {
		case i == 1:
			name = "  SYNDICATION " // nama sama dengan raksasa setelah normalisasi
		case i >= 2 && i <= 4:
			name = budi[i-2]
		case i >= 5 && i <= 54:
			name = "Admin" // 50 id, melewati ambang ambigu N=20
		default:
			name = fmt.Sprintf("Penulis %06d", i)
		}
		rows = append(rows, []any{id, name})
		tailIDs = append(tailIDs, id)
	}
	return rows, tailIDs
}

type source struct {
	g       *gen
	tail    []string
	n, i    int
	start   time.Time
	step    int64 // nanodetik per baris
	prev    time.Time
	started time.Time
}

func (s *source) Next() bool { s.i++; return s.i <= s.n }
func (s *source) Err() error { return nil }

func (s *source) Values() ([]any, error) {
	idx := s.i - 1
	g := s.g

	t := s.start.Add(time.Duration(int64(idx)*s.step + g.r.Int63n(s.step))).Truncate(time.Microsecond)
	if idx > 0 && g.r.Float64() < dupTimeShare {
		t = s.prev
	}
	s.prev = t
	id := ulid.MustNew(ulid.Timestamp(t), g.ent).String()

	author := giantID
	if g.r.Float64() >= giantShare {
		author = s.tail[g.r.Intn(len(s.tail))]
	}

	lang := pickLang(g.r)
	title := g.text(4, 9, lang)
	body := g.text(30, 80, lang)

	// Kata uji dengan frekuensi pasti (deterministik berdasarkan nomor baris).
	if idx%5 < 2 {
		body += " berita" // umum: 40%
	}
	if idx%100 == 13 {
		body += " kebijakan" // menengah: 1%
	}
	if idx%2000 == 777 {
		body += " anggaran" // menengah-rendah: 0,05%
	}
	if idx%100000 == 7 {
		body += " hidrokarbon" // jarang: 1 per 100 ribu baris
	}

	if idx > 0 && idx%100000 == 0 {
		slog.Info("progres", "baris", idx, "berjalan", time.Since(s.started).Round(time.Second))
	}
	return []any{id, author, title, body, t, t}, nil
}

func fatal(msg string, err error) {
	slog.Error(msg, "err", err)
	os.Exit(1)
}

func main() {
	n := flag.Int("n", 200000, "jumlah artikel")
	seed := flag.Int64("seed", 42, "seed acak (hasil bisa diulang)")
	reset := flag.Bool("reset", false, "TRUNCATE tabel dulu (HANYA untuk database lokal)")
	flag.Parse()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fatal("DATABASE_URL kosong", nil)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fatal("gagal membuat pool", err)
	}
	defer pool.Close()

	var existing int64
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM authors").Scan(&existing); err != nil {
		fatal("gagal membaca authors (sudah migrate-up?)", err)
	}
	if existing > 0 {
		if !*reset {
			fatal("tabel sudah berisi data, jalankan dengan -reset", nil)
		}
		if _, err := pool.Exec(ctx, "TRUNCATE articles, authors"); err != nil {
			fatal("gagal TRUNCATE", err)
		}
	}

	started := time.Now()
	g := newGen(*seed)

	authorRows, tailIDs := buildAuthors(*n)
	if _, err := pool.CopyFrom(ctx, pgx.Identifier{"authors"}, []string{"id", "name"},
		pgx.CopyFromRows(authorRows)); err != nil {
		fatal("COPY authors gagal", err)
	}
	slog.Info("authors masuk", "jumlah", len(authorRows))

	span := time.Duration(spanDays) * 24 * time.Hour
	step := int64(span) / int64(*n)
	if step < 1 {
		step = 1
	}
	src := &source{
		g: g, tail: tailIDs, n: *n,
		start: time.Now().UTC().Add(-span), step: step, started: started,
	}
	cols := []string{"id", "author_id", "title", "body", "created_at", "updated_at"}
	copied, err := pool.CopyFrom(ctx, pgx.Identifier{"articles"}, cols, src)
	if err != nil {
		fatal("COPY articles gagal", err)
	}
	slog.Info("selesai", "artikel", copied, "durasi", time.Since(started).Round(time.Second))
}
