<context>
# Overview  
Portctl is a cross-platform CLI tool for managing, listing, and killing processes bound to network ports. It targets developers and system administrators who need to resolve port conflicts, automate cleanup, and ensure safe process management. The tool emphasizes safety, verifiability, and robust error handling.

# Core Features  
- List processes by port, user, or service type
- Kill processes by port, PID, user, or service type, with confirmation and force options
- Scan ports and grab service banners
- Watch ports in real-time with notifications
- Config management with reset/edit and safety prompts
- gRPC service stubs for future extensibility
- Comprehensive test coverage (unit, integration, BDD)
- CI/CD pipeline with TDD, BDD, security, and vulnerability checks

# User Experience  
- Developer and DevOps personas
- CLI-first workflow, with clear prompts and error messages
- Confirmation for destructive actions, with --yes override
- Output suitable for scripting and automation
- Real-time feedback for watch/scan commands
</context>
<PRD>
# Technical Architecture  
- Go 1.21+, cross-platform (macOS, Linux, Windows)
- CLI built with Cobra, colored output (fatih/color), notifications (beeep)
- Process management via lsof/netstat/taskkill
- gRPC stubs for future API integration
- Config via Viper
- CI/CD: GitHub Actions, Makefile, GoReleaser

# Development Roadmap  
- Phase 1: Core CLI commands (list, kill, scan, watch)
- Phase 2: Safety features (confirmation, error handling, config reset)
- Phase 3: gRPC service stubs and integration
- Phase 4: Comprehensive TDD/BDD coverage
- Phase 5: CI/CD pipeline with security/vuln checks

# Logical Dependency Chain
- Foundation: Process listing and management
- Add: Kill, scan, and watch commands
- Integrate: Config management and safety features
- Extend: gRPC stubs and API
- Finalize: Test coverage and CI/CD

# Risks and Mitigations  
- Risk: Uncaught panics or runtime errors → Mitigation: TDD for all error paths
- Risk: Placeholder/legacy logic → Mitigation: Replace with robust, tested code
- Risk: User error in destructive actions → Mitigation: Confirmation prompts, --yes flag
- Risk: Gaps between docs and code → Mitigation: Doc-driven test cases
- Risk: Pipeline failures → Mitigation: CI/CD test for all steps

# Appendix  
- See README.md for CLI usage and safety features
- See arc42 docs for architecture constraints
</PRD>