Feature: Port management CLI
  As a developer
  I want to list and kill processes on ports
  So that I can resolve port conflicts and automate cleanup

  Scenario: List all processes with open ports
    When I run `portctl list`
    Then I should see a list of processes using ports

  Scenario: Kill process on a specific port
    Given a process is using port 8080
    When I run `portctl kill 8080 --yes`
    Then the process on port 8080 should be terminated
