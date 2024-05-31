Feature: Aborting Work
    Scenario: Aborting a waiting machine is ignored
        Given a system in state "waiting"
        When the system "abort" is requested
        Then the request is ignored
        And the system is found in state "waiting"

    Scenario: Aborting a busy machine
        Given a system in state "waiting"
        When the system "start" is requested
        And some work has progressed
        When the system "abort" is requested
        Then work is canceled
        And the system is found in state "waiting"
