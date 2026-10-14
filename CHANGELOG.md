# Changelog

## [0.4.0](https://github.com/infin-robotics/backplane/compare/v0.3.0...v0.4.0) (2026-10-14)


### ⚠ BREAKING CHANGES

* replace Graph.Mermaid() with mermaid.Render(graph), and Graph.MermaidWith(backplane.MermaidOptions{...}) with mermaid.RenderWith(graph, mermaid.Options{...}). Direction constants now live in mermaid without the Mermaid prefix.
* consumers must import github.com/infin-robotics/backplane. Public APIs and runtime behaviour are unchanged.

### Features

* construct Latest from channels ([c5c6361](https://github.com/infin-robotics/backplane/commit/c5c63613391545ee20bdcd32f258b499bd4287e2))
* move Mermaid rendering into its own package ([80fc296](https://github.com/infin-robotics/backplane/commit/80fc2963f32ae5c3b084297d999f1d88c75c56aa))


### Code Refactoring

* migrate backplane module to infin-robotics ([81adbec](https://github.com/infin-robotics/backplane/commit/81adbecdc7b71a1e43fef8549ffe41019f13b03f))

## [0.3.0](https://github.com/sunfish-robotics/backplane/compare/v0.2.0...v0.3.0) (2026-08-31)


### Features

* make Mermaid graphs easier to read ([260a895](https://github.com/sunfish-robotics/backplane/commit/260a895551e0c60b291acac3c413795cec9435a1))

## [0.2.0](https://github.com/sunfish-robotics/backplane/compare/v0.1.0...v0.2.0) (2026-08-24)


### Features

* complete topics when publishers close ([43d1dbd](https://github.com/sunfish-robotics/backplane/commit/43d1dbddf5ae601a271167b45f70825a94339773))


### Bug Fixes

* return caller cancellation after clean shutdown ([59b8599](https://github.com/sunfish-robotics/backplane/commit/59b8599cf0ee0cc9583a07a265abff68486457b6))

## 0.1.0 (2026-08-21)

Initial public release of Backplane, a small in-process application runtime for Go.

### Runtime

- Declare modules as ordinary `func(context.Context, ...dependencies) error` functions. Channel directions declare typed publishers and subscribers; other parameters declare caller-owned resources.
- Validate module signatures, topic producers, and resource bindings before any module starts. Concrete resources can satisfy interface parameters when the match is unambiguous.
- Run a fixed module set concurrently under one shared context. The first module error cancels its siblings, waits for them to return, and reports the failing module by name.

### Topics and state

- Fan values from multiple publishers to every live subscriber of the same exact Go type. The topic pump accepts at most one value at a time, so slow subscribers apply backpressure to subsequent publications.
- Close subscriptions after every publisher has returned, remove subscriptions whose module has already stopped, and drain and drop publications during cancellation so blocked publishers can unwind.
- Provide `Latest[T]` for current-state consumers, with timestamped `Load` access and non-backpressuring, latest-wins `Watch` notifications.

### Topology and documentation

- Derive a deterministic graph of modules, resources, and topics from the same declarations used at runtime, without starting modules or constructing resources.
- Render the topology as a Mermaid flowchart.
- Document the lifecycle, ownership, and delivery contracts with package documentation and executable examples.
