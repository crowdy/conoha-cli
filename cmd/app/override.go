package app

import "fmt"

// composeOverride returns a compose override document (YAML) that:
//   - pins the web service's container name to <project>-<slot>-<web>
//   - maps 127.0.0.1:0:<port> so the kernel picks a free host port
//   - when hasAccessories, joins the <app>-accessories_default network so the
//     web container can resolve and reach accessory services (db/cache/etc.)
//
// It does not touch any other service; accessories keep their compose-declared
// container_name and ports.
func composeOverride(app, slot, webService string, webPort int, hasAccessories bool) string {
	base := fmt.Sprintf(`services:
  %[3]s:
    container_name: %[1]s-%[2]s-%[3]s
    ports:
      - "127.0.0.1:0:%[4]d"
`, app, slot, webService, webPort)
	if !hasAccessories {
		return base
	}
	return base + fmt.Sprintf(`    networks:
      - default
      - accessories
networks:
  accessories:
    name: %[1]s-accessories_default
    external: true
`, app)
}

// slotProjectName is the compose -p value for a blue/green slot.
func slotProjectName(app, slot string) string {
	return fmt.Sprintf("%s-%s", app, slot)
}

// accessoryProjectName is the compose -p value for the persistent accessory stack.
func accessoryProjectName(app string) string {
	return app + "-accessories"
}
