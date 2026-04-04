// MIT License
//
// Copyright (c) 2026-present adachng (github.com/adachng)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package proxy

import (
	"net"
)

// Interface for events. The methods are made to be near-impossible to mutate the [App] instance.
//
// Implementations must not attempt to mutate [App] instance within methods in [EventListener].
//
// Note that lAddr stands for local address and rAddr stands for remote address.
type EventListener interface {
	// Any non-nil error results in calling this method with a few execeptions.
	// Client code may ignore [ErrLRepeatedClose], [ErrCRepeatedClose], and [ErrPeerClose].
	//
	// Does not get called if the scenario has another relevant method.
	// For example, if [net.Listen] resulted in an error, this will not
	// get called since the error is passed into [EventListener.AttemptedListen] instead.
	GotError(uuid string, err error)

	// Called only once after the [net.Listen] call.
	AttemptedListen(lAddr net.Addr, err error)

	// Called each time after [net.Accept] is called.
	AttemptedAccept(lAddr net.Addr, rAddr net.Addr, err error)

	// Called after failing validation of the inbound connection's remote address.
	FailedInbConn(rAddr net.Addr, match string)

	// Called after successful validation of the inbound connection's remote address.
	ValidatedInbConn(rAddr net.Addr, match string)

	// Called each time after [net.Dial] is called with the outbound connection's local and remote address as input.
	AttemptedDial(lAddr net.Addr, rAddr net.Addr, err error)

	// Called after successfully establishing connection pair after sucessful validation of inbound
	// connection's remote address and successful outbound connection.
	GotConnPair(uuid string, inbLAddr net.Addr, srcRAddr net.Addr, outbLAddr net.Addr, dstRAddr net.Addr)

	// Called via the [io.TeeReader]'s [io.Writer] implementation.
	RelayedBytes(uuid string, b []byte, srcRAddr net.Addr, dstRAddr net.Addr)

	// Called after [io.Copy] returns. The source connection referred here
	// is the source of the bytes in the copy, not the inbound connection.
	AttemptedIOCopy(uuid string, bytesWritten int64, err error, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr)

	// Called each time after [net.Conn.Close] is called. May be associated with UUID (ignore if == "").
	ClosedConn(uuid string, lAddr net.Addr, rAddr net.Addr, err error)

	// Called each time after [net.Listener.Close] is called (which is only once).
	ClosedListener(lAddr net.Addr, err error)
}

// Implements [io.Writer] for [io.TeeReader] to log all bytes relayed in hex.
type hexWriter struct {
	relayedBytesEvent func() func(uuid string, b []byte, srcAddr net.Addr, dstAddr net.Addr) // function that returns a function for concurrent-safe proxy event subscriber
	uuid              string                                                                 // UUID of the inbound and outbound connection pair

	srcAddr net.Addr // the remote address of the source of the bytes (may either be the inbound or outbound connection)
	dstAddr net.Addr // the remote address of the destination of the bytes (may either be the inbound or outbound connection)
}

func newHexWriter(
	relayedBytesEvent func() func(uuid string, b []byte, srcAddr net.Addr, dstAddr net.Addr),
	uuid string,
	srcA net.Addr,
	dstA net.Addr,
) hexWriter {
	return hexWriter{
		relayedBytesEvent: relayedBytesEvent,
		uuid:              uuid,

		srcAddr: srcA,
		dstAddr: dstA,
	}
}

func (h hexWriter) Write(b []byte) (n int, err error) {
	if h.relayedBytesEvent != nil && h.relayedBytesEvent() != nil {
		h.relayedBytesEvent()(h.uuid, b, h.srcAddr, h.dstAddr)
	}

	return len(b), nil
}

func (a *App) gotError(uuid string, err error) {
	if a.subscriber != nil {
		a.subscriber.GotError(uuid, err)
	}
}

func (a *App) attemptedListen(lAddr net.Addr, err error) {
	if a.subscriber != nil {
		a.subscriber.AttemptedListen(lAddr, err)
	}
}

func (a *App) attemptedAccept(lAddr net.Addr, rAddr net.Addr, err error) {
	if a.subscriber != nil {
		a.subscriber.AttemptedAccept(lAddr, rAddr, err)
	}
}

func (a *App) failedInbConn(rAddr net.Addr, match string) {
	if a.subscriber != nil {
		a.subscriber.FailedInbConn(rAddr, match)
	}
}

func (a *App) validatedInbConn(rAddr net.Addr, match string) {
	if a.subscriber != nil {
		a.subscriber.ValidatedInbConn(rAddr, match)
	}
}

func (a *App) attemptedDial(lAddr net.Addr, rAddr net.Addr, err error) {
	if a.subscriber != nil {
		a.subscriber.AttemptedDial(lAddr, rAddr, err)
	}
}

func (a *App) gotConnPair(uuid string, inbLAddr net.Addr, srcRAddr net.Addr, outbLAddr net.Addr, dstRAddr net.Addr) {
	if a.subscriber != nil {
		a.subscriber.GotConnPair(uuid, inbLAddr, srcRAddr, outbLAddr, dstRAddr)
	}
}

func (a *App) attemptedIOCopy(uuid string, bytesWritten int64, err error, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr) {
	if a.subscriber != nil {
		a.subscriber.AttemptedIOCopy(uuid, bytesWritten, err, srcLAddr, srcRAddr, dstLAddr, dstRAddr)
	}
}

func (a *App) closedConn(uuid string, lAddr net.Addr, rAddr net.Addr, err error) {
	if a.subscriber != nil {
		a.subscriber.ClosedConn(uuid, lAddr, rAddr, err)
	}
}

func (a *App) closedListener(lAddr net.Addr, err error) {
	if a.subscriber != nil {
		a.subscriber.ClosedListener(lAddr, err)
	}
}
