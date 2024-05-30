Feature: Wrong transitions

    Scenario: Aborting a waiting machine
        Given a machine in state "waiting"
        When the "abort" command is cast
        Then the following events can be observed:
            | RequestIgnored |
