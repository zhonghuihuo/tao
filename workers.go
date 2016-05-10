/* Worker pool is a pool of go-routines running for executing callbacks,
each client's message handler is permanently hashed into one specified
worker to execute, so it is in-order for each client's perspective. */
package tao

type WorkerPool struct {
  workers []*worker
  closeChan chan struct{}
}

func NewWorkerPool(vol int) *WorkerPool {
  if vol <= 0 {
    vol = WORKERS
  }

  pool := &WorkerPool{
    workers: make([]*worker, vol),
    closeChan: make(chan struct{}),
  }

  for i, _ := range pool.workers {
    pool.workers[i] = newWorker(i, 1024, pool.closeChan)
    if pool.workers[i] == nil {
      panic("worker nil")
    }
  }

  return pool
}

func (wp *WorkerPool) Put(k interface{}, cb func()) error {
  var code uint32
  var err error
  if code, err = hashCode(k); err != nil {
    return err
  }
  return wp.workers[code & uint32(len(wp.workers) - 1)].put(workerFunc(cb))
}

func (wp *WorkerPool) Close() {
  close(wp.closeChan)
}

type worker struct {
  index int
  callbackChan chan workerFunc
  closeChan chan struct{}
}

func newWorker(i int, c int, closeChan chan struct{}) *worker {
  w := &worker{
    index: i,
    callbackChan: make(chan workerFunc, c),
    closeChan: closeChan,
  }
  go w.start()
  return w
}

func (w *worker) start() {
  for {
    select {
    case <-w.closeChan:
      break
    case cb := <-w.callbackChan:
        cb()
    }
  }
  close(w.callbackChan)
}

func (w *worker) put(cb workerFunc) error {
  select {
  case w.callbackChan<- cb:
    return nil
  default:
    return ErrorWouldBlock
  }
}
