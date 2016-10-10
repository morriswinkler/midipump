# midipump

Midipump is a program to control peristaltic pumps through midi. The backend is written in go and works on a raspbery pi.
To port to an other device see serial.go, as it holds all the device specific code for the midi serial initialisation.