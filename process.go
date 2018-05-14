package mutual

import (
	"container/heap"
	"fmt"
	"sync"
)

type process struct {
	rwmu         sync.RWMutex
	me           int
	clock        *clock
	chans        []chan *message
	requestQueue rpq

	resource *resource

	sentTime       []int // 最近一次给别的 process 发送的消息，所携带的最后时间
	receiveTime    []int // 最近一次从别的 process 收到的消息，所携带的最后时间
	minReceiveTime int   // lastReceiveTime 中的最小值

	toCheckRule5Chan chan struct{} // 每次收到 message 后，都靠这个 chan 来通知检查此 process 是否已经满足 rule 5，以便决定是否占有 resource

	requestChan chan struct{}
	releaseChan chan struct{}
	occupyChan  chan struct{}

	// TODO: 删除此处内容
	isOccupying bool
}

func newProcess(me int, r *resource, chans []chan *message) *process {
	p := &process{
		me:             me,
		resource:       r,
		clock:          newClock(),
		chans:          chans,
		requestQueue:   make(rpq, 0, 1024),
		sentTime:       make([]int, len(chans)),
		receiveTime:    make([]int, len(chans)),
		minReceiveTime: 0,

		toCheckRule5Chan: make(chan struct{}),
		requestChan:      make(chan struct{}),
		occupyChan:       make(chan struct{}),
	}

	eventLoop(p)

	debugPrintf("[%d]P%d 完成创建工作", p.clock.getTime(), p.me)

	return p
}

func (p *process) updateMinReceiveTime() {
	i := (p.me + 1) % len(p.chans)
	minTime := p.receiveTime[i]
	for i, t := range p.receiveTime {
		if i == p.me {
			continue
		}
		minTime = min(minTime, t)
	}
	p.minReceiveTime = minTime
}

// TODO: finish this
type sendMsg struct {
	receiveID int
	msg       *message
}

func (p *process) handleSend(sm *sendMsg) {
	sm.msg.timestamp = p.clock.getTime()

	debugPrintf("[%d]P%d -> P%d，消息内容 %s", p.clock.getTime(), p.me, sm.receiveID, sm.msg)

	p.sentTime[sm.receiveID] = max(p.sentTime[sm.receiveID], p.clock.getTime())

	// go func() {
	// 	p.chans[sm.receiveID] <- sm.msg
	// }()

	p.chans[sm.receiveID] <- sm.msg

}

func (p *process) push(r *request) {
	heap.Push(&p.requestQueue, r)
	debugPrintf("[%d]P%d push(%s) 后，request queue %v", p.clock.getTime(), p.me, r, p.requestQueue)
}

func (p *process) pop(r *request) {
	req := heap.Pop(&p.requestQueue).(*request)
	if req != r {
		msg := fmt.Sprintf("需要删除的是 %s，实际删除的是 %s，P%d.RQ%s", r, req, p.me, p.requestQueue)
		panic(msg)
	}

	debugPrintf("[%d]P%d pop(%s) 后，request queue %v", p.clock.getTime(), p.me, req, p.requestQueue)
}

// func (p *process) request() {
// 	debugPrintf("[%d]P%d 准备 request", p.clock.getTime(), p.me)
// 	p.requestChan <- struct{}{}
// }
