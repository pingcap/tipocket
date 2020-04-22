package porcupine

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/anishathalye/porcupine"

	"github.com/pingcap/tipocket/pkg/core"
)

// Checker is a linearizability checker powered by Porcupine.
type Checker struct{}

// Check checks the history of operations meets liearizability or not with model.
// False means the history is not linearizable.
func (Checker) Check(m core.Model, ops []core.Operation) (bool, error) {
	pModel := porcupine.Model{
		Init:  m.Init,
		Step:  m.Step,
		Equal: m.Equal,
	}
	events, err := ConvertOperationsToEvents(ops)
	if err != nil {
		return false, err
	}
	log.Printf("begin to verify %d events", len(events))
	res, info := porcupine.CheckEventsVerbose(pModel, events, 0)
	if res != porcupine.Ok {
		file, err := ioutil.TempFile("./", "*.html")
		if err != nil {
			log.Fatalf("failed to create temp file")
		}
		err = porcupine.Visualize(pModel, info, file)
		if err != nil {
			log.Fatalf("visualization failed")
		}
		log.Printf("wrote visualization to %s", file.Name())
		return false, nil
	}
	return true, nil
}

// Name is the name of porcupine checker
func (Checker) Name() string {
	return "porcupine_checker"
}

// ConvertOperationsToEvents converts core.Operations to porcupine.Event.
func ConvertOperationsToEvents(ops []core.Operation) ([]porcupine.Event, error) {
	if len(ops)%2 != 0 {
		return nil, fmt.Errorf("history is not complete")
	}

	procID := map[int64]int{}
	id := int(0)
	events := make([]porcupine.Event, 0, len(ops))
	for _, op := range ops {
		if op.Action == core.InvokeOperation {
			event := porcupine.Event{
				ClientId: int(op.Proc),
				Kind:     porcupine.CallEvent,
				Id:       id,
				Value:    op.Data,
			}
			events = append(events, event)
			procID[op.Proc] = id
			id++
		} else if op.Action == core.ReturnOperation {
			if op.Data == nil {
				continue
			}

			matchID := procID[op.Proc]
			delete(procID, op.Proc)
			event := porcupine.Event{
				ClientId: int(op.Proc),
				Kind:     porcupine.ReturnEvent,
				Id:       matchID,
				Value:    op.Data,
			}
			events = append(events, event)
		}
	}

	if len(procID) != 0 {
		return nil, fmt.Errorf("history is not complete")
	}

	return events, nil
}
