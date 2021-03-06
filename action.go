package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"gopkg.in/Shopify/sarama.v1"
)

func HttpStatConsumerAction(w http.ResponseWriter, r *http.Request) {
	type consumerStat struct {
		Topic string
		Url   string
	}

	managers := server.managers
	var res []consumerStat
	for _, mgr := range managers {
		res = append(res, consumerStat{Topic: mgr.Topic, Url: mgr.Url})
	}

	echo2client(w, r, res)
}

func HttpStatWorkerAction(w http.ResponseWriter, r *http.Request) {
	type workerStat struct {
		Topic     string
		Url       string
		Partition int32
		UUID      string
		Offset    int64
	}
	var res []workerStat
	for _, mgr := range server.managers {
		for _, worker := range mgr.workers {
			offset, err := worker.Consumer.OffsetManager().Offsets(worker.Topics[0])
			if err != nil {
				continue
			}
			for k, v := range offset {
				res = append(res, workerStat{
					Topic:     worker.Topics[0],
					Url:       worker.Callback.Url,
					Partition: k,
					UUID:      worker.Consumer.Instance().ID,
					Offset:    v,
				})
			}
		}
	}

	echo2client(w, r, res)
}

func HttpAdminSkipAction(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	topic := query.Get("topic")
	partition := query.Get("partition")
	group := query.Get("group")
	offset := query.Get("offset")

	if topic == "" || partition == "" || group == "" || offset == "" {
		echo2client(w, r, "invalid param")
		return
	}

	i_partition, _ := strconv.Atoi(partition)
	i_offset, _ := strconv.ParseInt(offset, 10, 64)

	mgr, err := server.Find(topic, group)
	if err != nil {
		echo2client(w, r, "invalid topic/group")
		return
	}

	worker, err := mgr.Find(i_partition)
	if err != nil {
		echo2client(w, r, "invalid topic/partition")
		return
	}

	msg := &sarama.ConsumerMessage{
		Topic:     topic,
		Partition: int32(i_partition),
		Offset:    i_offset,
	}
	worker.Consumer.CommitUpto(msg)
	worker.Close()
	echo2client(w, r, "")
}

func echo2client(w http.ResponseWriter, r *http.Request, data interface{}) {
	res := map[string]interface{}{
		"errno":  0,
		"errmsg": "success",
		"data":   data,
	}

	if b, e := json.Marshal(res); e != nil {
		log.Printf("marshal http response error, %v", e)
	} else {
		io.WriteString(w, string(b))
	}

	return
}
