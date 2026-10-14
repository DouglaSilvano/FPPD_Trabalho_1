// V0 - Versão sequencial (baseline).
// Base: Código dado no moodle:


package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"
)


// Parâmetros do experimento

var cfg struct {
	n          int           // número de filósofos
	ciclos     int           // ciclos por filósofo: pensar -> adquirir -> comer -> liberar
	pensar     time.Duration // tempo simulado pensando
	comer      time.Duration // tempo simulado comendo
	silencioso bool          // não imprime a tabela caso for true
}

func registrarFlags() {
	flag.IntVar(&cfg.n, "n", 5, "número de filósofos")
	flag.IntVar(&cfg.ciclos, "ciclos", 20, "ciclos por filósofo")
	flag.DurationVar(&cfg.pensar, "pensar", 20*time.Millisecond, "tempo simulado pensando")
	flag.DurationVar(&cfg.comer, "comer", 30*time.Millisecond, "tempo simulado comendo")
	flag.BoolVar(&cfg.silencioso, "q", false, "silencioso: não imprime a tabela T/E (usado nas medições)")
}

func validarFlags() {
	if cfg.n < 2 || cfg.n > 32 {
		log.Fatalf("n deve estar entre 2 e 32 (recebido %d)", cfg.n)
	}
	if cfg.ciclos < 1 {
		log.Fatalf("ciclos deve ser >= 1 (recebido %d)", cfg.ciclos)
	}
}

// Tabela de saída: uma coluna por filósofo

func cabecalho() {
	if cfg.silencioso {
		return
	}
	cols := make([]string, cfg.n)
	for i := range cols {
		cols[i] = fmt.Sprintf("[P%d]", i+1)
	}
	fmt.Printf("\n%s\n\n", strings.Join(cols, " "))
}

func tabela(id int, formato string, a ...any) {
	if cfg.silencioso {
		return
	}
	fmt.Printf("%s %s\n", strings.Repeat(" ", id*6), fmt.Sprintf(formato, a...))
}

// Estatísticas

type estatistica struct {
	ciclos      atomic.Int64 // ciclos concluídos
	esperas     atomic.Int64 // quantas esperas foram medidas
	esperaTotal atomic.Int64 // soma das esperas (ns)
	esperaMax   atomic.Int64 // maior espera (ns)
	concluiuEm  atomic.Int64 // instante (ns desde o início) em que terminou o último ciclo
}

var (
	stats      []estatistica
	comendo    atomic.Int64
	maxComendo atomic.Int64
	t0         time.Time // início do jantar
)

func atualizaMax(a *atomic.Int64, v int64) {
	for {
		atual := a.Load()
		if v <= atual || a.CompareAndSwap(atual, v) {
			return
		}
	}
}

func inicioEspera() time.Time { return time.Now() }

func fimEspera(id int, inicio time.Time) {
	d := int64(time.Since(inicio))
	e := &stats[id]
	e.esperaTotal.Add(d)
	e.esperas.Add(1)
	atualizaMax(&e.esperaMax, d)
}

func comecaComer() { atualizaMax(&maxComendo, comendo.Add(1)) }

func terminaComer(id int) {
	comendo.Add(-1)
	stats[id].ciclos.Add(1)
}

func concluiu(id int) { stats[id].concluiuEm.Store(int64(time.Since(t0))) }

func ms(ns float64) float64 { return ns / 1e6 }

// relatorio imprime o resumo, a tabela por filósofo e uma linha "CSV,..." usada pelo run_all.sh.
func relatorio(versao string, total time.Duration) {
	fmt.Printf("\n==== %s ====\n", versao)
	fmt.Printf("Tempo total ................... %s\n", total)
	fmt.Printf("Máx. filósofos comendo juntos . %d\n", maxComendo.Load())

	fmt.Println("\nEstatísticas por filósofo:")
	fmt.Println("Filósofo | Ciclos | Espera média (ms) | Maior espera (ms) | Concluiu em (s)")
	fmt.Println("---------+--------+-------------------+-------------------+----------------")
	var somaEspera, qtd, picoEspera, somaCiclos, ultimo int64
	ciclosCSV := make([]string, len(stats))
	for i := range stats {
		e := &stats[i]
		n, soma, pico, fim := e.esperas.Load(), e.esperaTotal.Load(), e.esperaMax.Load(), e.concluiuEm.Load()
		media := 0.0
		if n > 0 {
			media = float64(soma) / float64(n)
		}
		fmt.Printf("P%-7d | %6d | %17.3f | %17.3f | %15.3f\n",
			i+1, e.ciclos.Load(), ms(media), ms(float64(pico)), float64(fim)/1e9)
		somaEspera += soma
		qtd += n
		somaCiclos += e.ciclos.Load()
		if pico > picoEspera {
			picoEspera = pico
		}
		if fim > ultimo {
			ultimo = fim
		}
		ciclosCSV[i] = fmt.Sprint(e.ciclos.Load())
	}
	mediaGeral := 0.0
	if qtd > 0 {
		mediaGeral = float64(somaEspera) / float64(qtd)
	}
	fmt.Println("---------+--------+-------------------+-------------------+----------------")
	fmt.Printf("Geral    | %6d | %17.3f | %17.3f | %15.3f\n",
		somaCiclos, ms(mediaGeral), ms(float64(picoEspera)), float64(ultimo)/1e9)

	fmt.Printf("CSV,%s,%.3f,%.3f,%.3f,%d,%s\n",
		versao, ms(float64(total)), ms(mediaGeral), ms(float64(picoEspera)),
		maxComendo.Load(), strings.Join(ciclosCSV, "/"))
}

// V0: execução sequencial

func filosofo(id, ciclo int) {
	// pensar
	tabela(id, "T%d", ciclo)
	time.Sleep(cfg.pensar)

	// adquirir recursos: garfos estão sempre livres
	inicio := inicioEspera()
	fimEspera(id, inicio)

	// comer
	comecaComer()
	tabela(id, "E%d", ciclo)
	time.Sleep(cfg.comer)
	terminaComer(id)

	if ciclo == cfg.ciclos {
		concluiu(id)
	}
}

func main() {
	registrarFlags()
	flag.Parse()
	validarFlags()
	stats = make([]estatistica, cfg.n)

	cabecalho()

	t0 = time.Now()
	for c := 1; c <= cfg.ciclos; c++ {
		for j := 0; j < cfg.n; j++ {
			filosofo(j, c)
		}
	}
	total := time.Since(t0)

	relatorio("V0", total)
	teorico := time.Duration(cfg.ciclos*cfg.n) * (cfg.pensar + cfg.comer)
	fmt.Printf("Tempo teórico (ciclos*filósofos*(pensar+comer)) = %s\n", teorico)
}
