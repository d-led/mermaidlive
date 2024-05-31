Feature: Normal operation
    Scenario: Running the machine to completion
        Given a system in state "waiting"
        When the system "start" is requested
        Then some work has progressed
        And work is completed
        And the system is found in state "waiting"
