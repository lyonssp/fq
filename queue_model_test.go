package queue

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/commands"
	"github.com/leanovate/gopter/gen"
	"github.com/stretchr/testify/assert"
)

const testNamespace = "test"

func TestQueueModel(t *testing.T) {
	assert := assert.New(t)

	test := &commands.ProtoCommands{
		NewSystemUnderTestFunc: func(initialState commands.State) commands.SystemUnderTest {
			f, err := ioutil.TempFile("", "queue-*")
			assert.Nil(err)

			return &queueController{
				f:     f,
				queue: NewQueue(f),
			}
		},
		InitialStateGen: gen.Const(makeQueueModel()),
		InitialPreConditionFunc: func(_ commands.State) bool {
			return true
		},
		GenCommandFunc: func(st commands.State) gopter.Gen {
			return gen.Weighted([]gen.WeightedGen{
				{45, genPushCommand},
				{45, genPopCommand(st)},
				{10, genCrashCommand},
			})
		},
	}

	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.Property("model", commands.Prop(test))
	properties.TestingRun(t)
}

func genPushCommand(params *gopter.GenParameters) *gopter.GenResult {
	return gopter.NewGenResult(
		pushCommand{
			x: []byte(gen.Identifier()(params).Result.(string)),
		},
		gopter.NoShrinker,
	)
}

var genPopCommand = func(st commands.State) gopter.Gen {
	return func(params *gopter.GenParameters) *gopter.GenResult {
		return gopter.NewGenResult(
			popCommand{},
			gopter.NoShrinker,
		)
	}
}

func genCrashCommand(params *gopter.GenParameters) *gopter.GenResult {
	return gopter.NewGenResult(
		crashCommand{},
		gopter.NoShrinker,
	)
}

type pushCommand struct {
	x []byte
}

func (cmd pushCommand) Run(sut commands.SystemUnderTest) commands.Result {
	q := sut.(*queueController).queue
	err := q.Enqueue(cmd.x)
	if err != nil {
		return commands.Result(err)
	}
	return nil
}

func (cmd pushCommand) NextState(state commands.State) commands.State {
	st := state.(queueModel).clone()
	st.Push(cmd.x)
	return st
}

func (cmd pushCommand) PreCondition(_ commands.State) bool {
	return true
}

func (cmd pushCommand) PostCondition(st commands.State, result commands.Result) *gopter.PropResult {
	if e, ok := result.(error); ok {
		return &gopter.PropResult{Error: e}
	}

	return gopter.NewPropResult(true, "")
}

func (cmd pushCommand) String() string {
	return fmt.Sprintf("push(%s)", string(cmd.x))
}

type popCommand struct{}

func (cmd popCommand) Run(sut commands.SystemUnderTest) commands.Result {
	q := sut.(*queueController).queue
	front, err := q.Dequeue()
	if err != nil {
		return commands.Result(err)
	}
	return front
}

func (cmd popCommand) NextState(state commands.State) commands.State {
	st := state.(queueModel).clone()
	st.Pop()
	return st
}

func (cmd popCommand) PostCondition(st commands.State, result commands.Result) *gopter.PropResult {
	if e, ok := result.(error); ok {
		return &gopter.PropResult{Error: e}
	}

	got := result.([]byte)
	want := st.(queueModel).lastPopped
	if !bytes.Equal(got, want) {
		return gopter.NewPropResult(false, fmt.Sprintf("%s != %s", got, want))
	}

	return gopter.NewPropResult(true, "")
}

func (cmd popCommand) PreCondition(st commands.State) bool {
	return st.(queueModel).size() > 0
}

func (cmd popCommand) String() string {
	return "pop()"
}

type crashCommand struct{}

func (cmd crashCommand) Run(sut commands.SystemUnderTest) commands.Result {
	qc := sut.(*queueController)
	qc.crash()

	return nil
}

func (cmd crashCommand) NextState(state commands.State) commands.State {
	return state
}

func (cmd crashCommand) PostCondition(_ commands.State, result commands.Result) *gopter.PropResult {
	if e, ok := result.(error); ok {
		return &gopter.PropResult{Error: e}
	}
	return gopter.NewPropResult(true, "")
}

func (cmd crashCommand) PreCondition(st commands.State) bool {
	return true
}

func (cmd crashCommand) String() string {
	return "crash()"
}

var (
	_ commands.Command = pushCommand{}
	_ commands.Command = popCommand{}
	_ commands.Command = crashCommand{}
)

// queueController preserves the underlying reference to resources consumed by a
// Queue to enable commands that represent restarts, filesystem failures, etc.
type queueController struct {
	f     *os.File // file backing the queue
	queue *Queue   // queue under test
}

func (qc *queueController) crash() {
	qc.queue = NewQueue(qc.f)
}

// queueModel is an in-memory model of a FIFO queue
type queueModel struct {
	ls         []string
	lastPopped []byte
}

func makeQueueModel() queueModel {
	return queueModel{ls: make([]string, 0)}
}

func (mod *queueModel) Push(x []byte) error {
	mod.ls = append(mod.ls, string(x))
	return nil
}

func (mod *queueModel) Pop() ([]byte, error) {
	if len(mod.ls) <= 0 {
		return nil, errors.New("cannot pop from empty queue")
	}

	front := mod.ls[0]
	mod.lastPopped = make([]byte, len(front))
	copy(mod.lastPopped, front)
	mod.ls = mod.ls[1:]

	return []byte(front), nil
}

func (mod queueModel) size() int {
	return len(mod.ls)
}

func (mod queueModel) clone() queueModel {
	cp := make([]string, len(mod.ls))
	copy(cp, mod.ls)
	return queueModel{ls: cp, lastPopped: mod.lastPopped}
}

// flakyReadWriteSeeker is a io.ReadWriteSeeker middleware that
// can be used to fail the invocation of Read, Write, or Seek
// methods and otherwise delegates to an underlying
type flakyReadWriteSeeker struct {
	inner           io.ReadWriteSeeker
	readShouldFail  bool
	writeShouldFail bool
	seekShouldFail  bool
}

func newFlakyReadWriteSeeker(rws io.ReadWriteSeeker) *flakyReadWriteSeeker {
	return &flakyReadWriteSeeker{inner: rws}
}

func (rws *flakyReadWriteSeeker) Read(b []byte) (int, error) {
	if rws.readShouldFail {
		return 0, errors.New("Oh no!")
	}
	return rws.inner.Read(b)
}

func (rws *flakyReadWriteSeeker) Write(b []byte) (int, error) {
	if rws.writeShouldFail {
		return 0, errors.New("Oh no!")
	}
	return rws.inner.Write(b)
}

func (rws *flakyReadWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	if rws.seekShouldFail {
		return 0, errors.New("Oh no!")
	}
	return rws.inner.Seek(offset, whence)
}

func (rws *flakyReadWriteSeeker) failNextRead() {
	rws.readShouldFail = true
}

func (rws *flakyReadWriteSeeker) failNextWrite() {
	rws.writeShouldFail = true
}

func (rws *flakyReadWriteSeeker) failNextSeek() {
	rws.seekShouldFail = true
}

var _ io.ReadWriteSeeker = new(flakyReadWriteSeeker)
