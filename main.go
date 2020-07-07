// rrns_namecom project main.go
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/namedotcom/go/namecom"
)

var address = flag.String("bindaddress", "127.0.0.1:5553", "bind address for http")

type ddns_namecom struct {
	domain      string
	username    string
	token       string
	answer      string
	updateall   string
	recreate    string
	update_type string
	host        string
	delete_dup  string
	delete_all  string
	client      *namecom.NameCom
	records     []*namecom.Record
}

func main() {
	http.HandleFunc("/update", update_handler)

	http.ListenAndServe(*address, nil)
}

func update_handler(w http.ResponseWriter, r *http.Request) {
	ddns := &ddns_namecom{
		domain:      r.FormValue("domain"),
		username:    r.FormValue("username"),
		token:       r.FormValue("token"),
		answer:      r.FormValue("answer"),
		updateall:   r.FormValue("updateall"),
		recreate:    r.FormValue("recreate"),
		update_type: r.FormValue("type"),
		delete_dup:  r.FormValue("deletedup"),
		delete_all:  r.FormValue("deleteall"),
	}

	if ddns.update_type == "" {
		ddns.update_type = "A"
	}

	if ddns.update_type == "A" {
		if ddns.answer == "" {
			ddns.answer = r.Header.Get("X-Real-IP") // real IP if we use nginx
			if ddns.answer == "" {
				ddns.answer = r.Header.Get("X-FORWARDED-FOR")
			}
			if ddns.answer == "" {
				ddns.answer, _, _ = net.SplitHostPort(r.RemoteAddr)
			}
		}
	}

	log.Printf("Requst modify %s(%s) to %s from %s\n", ddns.domain, ddns.update_type, ddns.answer, r.RemoteAddr)

	if ddns.domain == "" || ddns.answer == "" {
		w.WriteHeader(503)
		return
	}

	if ddns.update_ddns() {
		w.Write([]byte("ok"))
	} else {
		w.WriteHeader(503)
	}
}

func (d *ddns_namecom) update_record() bool {
	var updated bool
	var matchedcounter = 0
	var updatedcounter = 0
	for _, record := range d.records {
		if record.Host == d.host && record.Type == d.update_type {
			matchedcounter += 1
			record.Answer = d.answer
			_, err := d.client.UpdateRecord(record)
			if err != nil {
				log.Printf("update record fail: %d, err:%s\n", record.ID, err.Error())
				if d.delete_dup == "1" {
					_, err := d.client.DeleteRecord(&namecom.DeleteRecordRequest{
						DomainName: d.domain,
						ID:         record.ID,
					})
					if err != nil {
						log.Printf("Delete dup record fail:%d err:%s\n", record.ID, err.Error())
					} else {
						log.Printf("Delete dup record success:%d err:%s\n", record.ID)
					}
				}
			} else {
				log.Printf("Update record success: %d\n", record.ID)
				updated = true
				updatedcounter += 1
				if d.updateall != "1" {
					return true
				}
			}
		}
	}
	if updated {
		log.Printf("Update records success/found/all: %d/%d/%d\n",
			updatedcounter, matchedcounter, len(d.records))
		return true
	} else {
		return false
	}
}

func (d *ddns_namecom) create_record() bool {
	var record = &namecom.Record{
		DomainName: d.domain,
		Host:       d.host,
		Answer:     d.answer,
		Type:       d.update_type,
	}
	record, err := d.client.CreateRecord(record)
	if err != nil {
		return false
	} else {
		log.Printf("Create record success: %#v\n", record.ID)
		return true
	}
}

func (d *ddns_namecom) delete_record() bool {
	var matchedcounter = 0
	var deletedcounter = 0
	for _, record := range d.records {
		if record.Host == d.host && record.Type == d.update_type {
			matchedcounter += 1
			_, err := d.client.DeleteRecord(&namecom.DeleteRecordRequest{
				DomainName: d.domain,
				ID:         record.ID,
			})
			if err != nil {
				log.Printf("Delete record fail id: %d, err:%s\n", record.ID)
			} else {
				deletedcounter += 1
				log.Printf("Delete record success id: %d\n", record.ID)
			}
		}
	}
	// always success
	log.Printf("Delete records success/found/all: %d/%d/%d\n",
		deletedcounter, matchedcounter, len(d.records))
	return true
}

func (d *ddns_namecom) update_ddns() bool {
	d.client = namecom.New(d.username, d.token)
	if d.client == nil {
		return false
	}
	index := strings.LastIndex(d.domain, ".")
	if index == -1 {
		return false
	}

	// default type is A record
	if d.update_type == "" {
		d.update_type = "A"
	}

	path := strings.Split(d.domain, ".")

	if len(path) < 2 {
		return false
	}
	d.host = strings.Join(path[0:len(path)-2], ".")
	d.domain = strings.Join(path[len(path)-2:len(path)], ".")

	log.Printf("%s %s", d.host, d.domain)

	resp, err := d.client.ListRecords(&namecom.ListRecordsRequest{
		DomainName: d.domain,
	})
	if err != nil {
		log.Printf("List records fail, err:%s\n", err.Error())
		return false
	}
	d.records = resp.Records
	log.Printf("%d records found\n", len(d.records))

	// if recreate then delete all record and create it again
	var trycreate bool
	if d.recreate == "1" || d.delete_all == "1" {
		log.Println("Try delete all record match this domain")
		if !d.delete_record() {
			log.Println("Delete record fail")
			return false
		}
	} else {
		if !d.update_record() {
			log.Println("Update record fail, try create it")
			trycreate = true
		}
	}
	if d.recreate == "1" || trycreate && d.delete_all != "1" {
		if !d.create_record() {
			log.Println("Create record fail, GG")
			return false
		}
	}
	return true
}
