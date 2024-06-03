@api
Feature: Concurrent access

    Scenario: two connected clients
        Given a system in state "waiting"
        And two connected clients
        When the system "start" is requested
        And the system is found in state "waiting"
        Then two clients have observed the same events
