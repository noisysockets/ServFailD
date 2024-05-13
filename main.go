// SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 The Noisy Sockets Authors.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package main // SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 The Noisy Sockets Authors.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

import (
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/miekg/dns"
)

var logger = slog.Default()

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	for _, q := range r.Question {
		logger.Info("Received DNS query",
			slog.Any("name", q.Name),
			slog.Any("type", dns.TypeToString[q.Qtype]))
	}

	m := &dns.Msg{}
	m.SetReply(r)
	m.SetRcode(r, dns.RcodeServerFailure)
	if err := w.WriteMsg(m); err != nil {
		logger.Warn("Failed to write DNS response", slog.Any("error", err))
	}
}

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen", ":5353", "Address to listen on")

	flag.Parse()

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Error("Failed to listen on TCP port", slog.Any("error", err))
		os.Exit(1)
	}
	defer lis.Close()

	pc, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		logger.Error("Failed to listen on UDP port", slog.Any("error", err))
		os.Exit(1)
	}
	defer pc.Close()

	srv := &dns.Server{
		Listener:   lis,
		PacketConn: pc,
	}

	dns.HandleFunc(".", handleDNSRequest)

	// Gracefully shutdown the server.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig

		logger.Info("Shutting down DNS server")

		if err := srv.Shutdown(); err != nil {
			logger.Error("Failed to shutdown DNS server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	logger.Info("Listening for DNS queries", slog.Any("address", lis.Addr().String()))

	if err := srv.ActivateAndServe(); err != nil {
		logger.Error("Failed to start DNS server", slog.Any("error", err))
		os.Exit(1)
	}
}
