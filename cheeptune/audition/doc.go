// Package audition is how a person listens to a folder of tunes and says
// which to keep.
//
// The design, written before the code. A library folder holds, for each
// tune, NAME.json (the score), NAME.cuts.json (the sheet), NAME.mp4 (a film
// of its piano roll with a line moving across it, with the tune as sound)
// and, once there is one, votes.json. Four pieces:
//
//   - Load reads a tune from the folder: the score, the sheet checked
//     against it, and the piece rendered with the sheet's voicing.
//   - Plate and Film draw the piece, with its parts marked and named as
//     they begin, and write the film with interfaces/movie.
//   - Page writes index.html: the list voted on. It talks to the
//     votes API when there is one, and keeps votes in the browser when it
//     is opened as a file.
//   - Handler serves the folder and the votes API; Serve binds it to the
//     loopback address only. A vote is a record in votes.json and moves
//     nothing on disk.
//
// What the server defends against: a web page in the listener's browser that
// posts or reads votes (a vote must be application/json, and Host must be
// localhost or 127.0.0.1 with the port). What it does not: any program on
// this machine, which can send any request; there is no login.
package audition
