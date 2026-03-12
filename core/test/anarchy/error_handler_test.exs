defmodule Anarchy.ErrorHandlerTest do
  use ExUnit.Case, async: true

  alias Anarchy.ErrorHandler

  describe "classify/1" do
    test "rate limiting is transient" do
      {class, recovery, _msg} = ErrorHandler.classify({:rate_limited, %{}})
      assert class == :transient
      assert recovery == :retry
    end

    test "unique violation is permanent" do
      error = %Postgrex.Error{postgres: %{code: :unique_violation}}
      {class, recovery, _msg} = ErrorHandler.classify(error)
      assert class == :permanent
      assert recovery == :fail
    end

    test "merge conflict needs escalation" do
      {class, recovery, _msg} = ErrorHandler.classify({:conflict, "file.ex"})
      assert class == :escalate
      assert recovery == :escalate
    end

    test "timeout is transient" do
      {class, recovery, _msg} = ErrorHandler.classify(:timeout)
      assert class == :transient
      assert recovery == :retry
    end

    test "cancelled is permanent skip" do
      {class, recovery, _msg} = ErrorHandler.classify(:cancelled)
      assert class == :permanent
      assert recovery == :skip
    end

    test "config error is permanent" do
      {class, recovery, _msg} = ErrorHandler.classify({:invalid_workflow_config, "bad"})
      assert class == :permanent
      assert recovery == :fail
    end

    test "unknown errors default to transient" do
      {class, recovery, _msg} = ErrorHandler.classify(:something_weird)
      assert class == :transient
      assert recovery == :retry
    end
  end

  describe "retriable?/1" do
    test "rate limit is retriable" do
      assert ErrorHandler.retriable?({:rate_limited, %{}})
    end

    test "unique violation is not retriable" do
      refute ErrorHandler.retriable?(%Postgrex.Error{postgres: %{code: :unique_violation}})
    end

    test "cancelled is not retriable" do
      refute ErrorHandler.retriable?(:cancelled)
    end
  end

  describe "backoff_ms/1" do
    test "first attempt is 10 seconds" do
      assert ErrorHandler.backoff_ms(0) == 10_000
    end

    test "exponential growth" do
      assert ErrorHandler.backoff_ms(1) == 20_000
      assert ErrorHandler.backoff_ms(2) == 40_000
      assert ErrorHandler.backoff_ms(3) == 80_000
    end

    test "caps at 5 minutes" do
      assert ErrorHandler.backoff_ms(10) == 300_000
      assert ErrorHandler.backoff_ms(100) == 300_000
    end
  end

  describe "handle/2" do
    test "returns :ok for retriable errors" do
      assert :ok = ErrorHandler.handle({:rate_limited, %{}}, context: "test")
    end

    test "returns error for permanent failures" do
      error = {:task_not_found, "abc"}
      assert {:error, ^error} = ErrorHandler.handle(error, context: "test")
    end

    test "returns escalation error for conflicts" do
      error = {:conflict, "file.ex"}
      assert {:error, {:needs_escalation, _msg}} = ErrorHandler.handle(error, context: "test")
    end
  end
end
