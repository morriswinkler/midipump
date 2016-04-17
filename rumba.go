package main

func rumba(cmd chan string) {

	port, err := openSerialRumba(rumbaDevice)
	if err != nil {
		Error.Printf("error opening rumba serial device: %s", err)
	}

	for {

		recv := <-cmd
		switch recv {
		case "home":
			_, err := port.WriteString("G28 Z")
			if err != nil {
				Error.Printf("error writing to rumba controller: %s", err)
			}
		case "move":
			_, err := port.WriteString("G1 Z50")
			if err != nil {
				Error.Printf("error writing to rumba controller: %s", err)
			}

		}

	}

}
