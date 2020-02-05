package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pingcap/tipocket/db/tidb"
	"github.com/pingcap/tipocket/pkg/check/porcupine"
	"github.com/pingcap/tipocket/pkg/model"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	historyFile = flag.String("history", "./history.log", "history file")
	names       = flag.String("names", "", "model names, seperate by comma")
	pprofAddr   = flag.String("pprof", "0.0.0.0:6060", "Pprof address")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		http.ListenAndServe(*pprofAddr, nil)
	}()

	go func() {
		<-sigs
		cancel()
	}()

	childCtx, cancel := context.WithCancel(ctx)

	go func() {
		for _, name := range strings.Split(*names, ",") {
			s := verify.Suit{}
			switch name {
			case "tidb_bank":
				s.Model, s.Parser, s.Checker = tidb.BankModel(), tidb.BankParser(), porcupine.Checker{}
			case "tidb_bank_tso":
				// Actually we can omit BankModel, since BankTsoChecker does not require a Model.
				s.Model, s.Parser, s.Checker = tidb.BankModel(), tidb.BankParser(), tidb.BankTsoChecker()
			//case "sequential":
			//	s.Parser, s.Checker = tidb.NewSequentialParser(), tidb.NewSequentialChecker()
			case "register":
				s.Model, s.Parser, s.Checker = model.RegisterModel(), model.RegisterParser(), porcupine.Checker{}
			case "":
				continue
			default:
				log.Printf("%s is not supported", name)
				continue
			}
			s.Verify(*historyFile)
		}

		cancel()
	}()

	<-childCtx.Done()
}
