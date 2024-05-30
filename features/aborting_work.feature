Feature: Aborting Work
    Scenario: Aborting a waiting machine is ignored
        Given the system is in state "waiting"
        When the system "abort" is requested
        Then the request is ignored

    Scenario: Aborting a busy machine
        Given the system is in state "waiting"
        When the system "start" is requested
        And some work has progressed
        When the system "abort" is requested
        Then work is canceled
