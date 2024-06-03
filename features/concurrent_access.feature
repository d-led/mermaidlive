@api
Feature: Concurrent access
    Scenario: two connected clients
        Given a system in state "waiting"
        And two connected clients
        When the system "start" is requested
        Then two clients have observed "Tick"
        Then two clients have observed "WorkDone"

    Scenario: cancellation seen by two clients
        Given a system in state "waiting"
        And two connected clients
        When the system "start" is requested
        And some work has progressed
        When the system "abort" is requested
        Then two clients have observed "WorkAbortRequested"
        Then two clients have observed "WorkAborted"
