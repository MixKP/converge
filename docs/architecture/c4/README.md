# C4 diagrams for SDD section 2.1

These files use PlantUML's bundled C4 library. Render them with a PlantUML installation that includes the standard library:

```text
plantuml *.puml
```

Use the diagrams in order: system context, containers, backend component/hexagonal view, then the code diagram.

The component diagram is intentionally based on the implemented dependencies. In particular, Scheduling uses Teacher and Commute through ports, while Branch and User are invoked independently by their HTTP handlers.

The code diagram is a focused package-and-port view of the backend. It is not a class diagram for every model; Chapter 3 remains the place for database details and per-feature method descriptions.
