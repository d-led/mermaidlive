Feature: Aborting Work

    Scenario: Aborting a waiting machine is ignored
        Given the system is in state "waiting"
        When the system "abort" is requested
        Then the request is ignored
