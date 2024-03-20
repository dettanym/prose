# Notes on correctness of OPA code

Ideally, independent of whether the default decision is allow or deny, the test-cases should all pass (i.e. correct allow / deny decision as expected). 
However, when you flip the default rule from ``default allow := false`` to ``default allow := true``, all test cases that should have failed (``test_invalid_*``), pass instead. Intuitively, this means that the code or test cases are not written correctly. However, this is not true, as explained below.   

```
allow if {
    #print(purpose_is_valid, purpose_is_allowed, processing_is_allowed)
    purpose_is_valid
    purpose_is_allowed
    processing_is_allowed
}
```

In this snippet, the rules ``purpose_is_valid``, ``purpose_is_allowed`` *and* ``processing_is_allowed``
all should be true (logical AND) for ``allow`` to be true. When OPA evaluates these rules for purposes that are not valid etc., the output is undefined rather than false. To check this, uncomment the line printing each of these values for the test cases. You will see that the values of at least one of these rules are undefined for the ``test_invalid_*`` cases. This is correct, expected behaviour of OPA rules: see examples for ``v {"hello" == "world"}`` or ``q["smoke2"]`` [here](https://www.openpolicyagent.org/docs/latest/policy-language/#the-basics).
So rules may evaluate to undefined.

Now, the OPA test command coerces undefined test results to false, see [here](
https://www.openpolicyagent.org/docs/latest/policy-testing/#test-results). 
So, *without* any default keywords, all tests, including the ``test_invalid_*`` ones with undefined allow result, evaluate to false. We don't want to rely on this ability as we need unambiguous true / false outputs for our OPA Envoy plugin or for others to use our OPA policies in a different context. 

The ``default`` keyword coerces a rule to either true or false. In this default rule, we set allow to false. So for the ``test_invalid_*`` test cases, each of those three sub-rules still evaluate to undefined, *but* the final allow rule evaluates to false.

So if we change the default allow to true, then for those test cases, *even though* the individual sub-rules still evaluate to undefined, the final allow rule evaluates to true, and so those test cases fail. 

We *could* set each of the sub-rules to be false, like this: ``default purpose_is_valid := false``. Then independent of whether a default rule is defined and what its value is, we would always get correct test results. However, then we would need to add a default statement for each new sub-rule, which is unfeasible, so we just stick to the standard of setting default to false. 

