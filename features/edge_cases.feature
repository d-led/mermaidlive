Feature: Edge cases
    @only
    Scenario: Trying to start a started machine
        Given a system in state "waiting"
        When the system "start" is requested
        Then some work has progressed
        And the system "start" is requested
        Then the request is ignored
        And work is completed
