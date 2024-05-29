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
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/sync/errgroup"
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

	mux := dns.NewServeMux()
	mux.HandleFunc(".", handleDNSRequest)

	pc, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		logger.Error("Failed to listen on UDP port", slog.Any("error", err))
		os.Exit(1)
	}
	defer pc.Close()

	udpServer := &dns.Server{
		Handler:    mux,
		PacketConn: pc,
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Error("Failed to listen on TCP port", slog.Any("error", err))
		os.Exit(1)
	}
	defer lis.Close()

	tcpServer := &dns.Server{
		Handler:  mux,
		Listener: lis,
	}

	g, ctx := errgroup.WithContext(context.Background())

	// Gracefully shutdown the server when a signal is received.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sig:
			return context.Canceled
		}
	})

	// We have to use multiple server instances as we can't serve both UDP and TCP
	// at the same time on the one server instance.
	for _, srv := range []*dns.Server{udpServer, tcpServer} {
		srv := srv

		g.Go(func() error {
			g.Go(func() error {
				<-ctx.Done()

				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := srv.ShutdownContext(shutdownCtx); err != nil {
					return err
				}

				return nil
			})

			return srv.ActivateAndServe()
		})
	}

	logger.Info("Listening for DNS queries (UDP/TCP)",
		slog.Any("address", lis.Addr().String()))

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Failed to serve DNS", slog.Any("error", err))
		os.Exit(1)
	}
}
