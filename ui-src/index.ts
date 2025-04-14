console.log(`loaded index.js`);

document.lastInput = "";
document.myReplica = null;

import { sourceReplicaIdKey } from "./common";

$(async function () {
  await reRenderGraph("waiting", "");

  console.log("done");

  while (true) {
    console.log("subscribing");
    try {
      await subscribeToEvents();
    } catch (err) {
      console.log("ERROR:", err?.message ?? err);
    }
    showDisconnectedAlert();
    console.log("waiting before reconnecting...");
    await sleep(5);
  }
});

async function subscribeToEvents() {
  await subscribe("/events", processEvent);
}

async function subscribe(
  streamUrl: string,
  processingFunc: (event: any) => Promise<void>,
) {
  let response = await fetch(streamUrl);

  if (response.status == 502) {
    // reconnect on timeout
    showDisconnectedAlert();
    await subscribe(streamUrl, processingFunc);
  } else if (response.status != 200) {
    // errored!
    showDisconnectedAlert();
    console.log("ERROR:", response.statusText);
    // reconnect
    await sleep(3);
    await subscribe(streamUrl, processingFunc);
  } else {
    hideDisconnectedAlert();
    flashConnectedAlert();
    // Get and show the message
    const reader = response?.body?.getReader();
    if (!reader) {
      console.log("ERROR: failed to read the messages");
    } else {
      // read all messages
      let currentMessage = "";
      while (true) {
        let chunk = await reader.read();
        if (chunk.done) {
          break;
        }
        currentMessage += new TextDecoder("utf-8").decode(chunk.value);
        let endlineAt = -1;
        do {
          endlineAt = currentMessage.indexOf("\n");
          if (endlineAt === -1) {
            break;
          }
          let messageToProcess = currentMessage.substring(0, endlineAt);
          currentMessage = currentMessage.substring(endlineAt + 1);
          try {
            let message = JSON.parse(messageToProcess);

            await processingFunc(message);
          } catch (err) {
            console.log("MESSAGE WAS:", messageToProcess);
            console.log("ERROR:", err?.message ?? err);
          }
        } while (true);
      }
    }
    // Call subscribe again to try to reconnect
    await sleep(1);
    await subscribe(streamUrl, processingFunc);
  }
}

function replaceText(selector, text: string) {
  $(selector).text(text);
}

function showLastError(text: string) {
  replaceText("#delayed-text", text);
}

function showLastEvent(text: string) {
  replaceText("#last-event", text);
}

function showVisitorsActive(count: number) {
  if (count == null) {
    return;
  }
  replaceText("#visitors-active", `${count}`);
}

function showVisitorsActiveInCluster(count: number) {
  if (count == null) {
    return;
  }
  replaceText("#visitors-active-cluster", `${count}`);
}

function showReplicasActive(msg: string) {
  if (msg == null) {
    return;
  }
  replaceText("#replicas", `${msg}`);
}

function showTotalVisitors(count: number) {
  if (count == null) {
    return;
  }
  replaceText("#total-visitors", `${count}`);
}

function showServerRevision(text: string) {
  replaceText("#server-revision", text);
}

function bindGraphClicks() {
  $("span.edgeLabel").wrap('<a href="#/"></a>');
  $("span.edgeLabel").on("click", function (e) {
    postCommand($(this).text());
  });
}

async function sleep(seconds: number) {
  await new Promise((resolve) => setTimeout(resolve, seconds * 1000 /*ms*/));
}

async function processEvent(event) {
  if (!event.name) {
    return;
  }

  console.log("INCOMING_EVENT:", event);

  let eventLine = formatEventIntoOneLine(event);

  switch (event.name) {
    case "WorkDone":
    case "WorkAborted":
      await reRenderGraph("waiting", "");
      break;
    case "WorkStarted":
      await reRenderGraph("working", `...`);
      break;
    case "LastSeenState":
      let state = `${event?.properties?.param}`;
      console.log(`rendering last seen state: ${state}`);
      await reRenderGraph(state, "");
      break;
    case "Tick":
      await reRenderGraph("working", ` ${event?.properties?.param}`);
      break;
    case "WorkAbortRequested":
      await reRenderGraph("aborting", "");
      break;
    case "RequestIgnored":
    case "CommandRejected":
      showLastError(eventLine);
      // do nothing
      break;
    case "ResourcesRefreshed":
      console.log("resources updated, reloading...");
      location.reload();
      break;
    case "VisitorsActive":
      showVisitorsActive(event?.properties?.param);
      // do not show this event in the log
      return;
    case "TotalClusterVisitorsActive":
      showVisitorsActiveInCluster(event?.properties?.param);
      // do not show this event in the log
      return;
    case "ReplicasActive":
      showReplicasActive(event?.properties?.param);
      // do not show this event in the log
      return;
    case "ConnectedToReplica":
      document.myReplica = event?.properties?.param;
      // do not show this event in the log
      return;
    case "TotalVisitors":
      showTotalVisitors(event?.properties?.param);
      // do not show this event in the log
      return;
    case "Revision":
      showServerRevision(event?.properties?.param);
      return;
    default:
      console.log(`unhandled event: ${event.name}`);
      // await reRenderGraph("", "");
      break;
  }

  showLastEvent(eventLine);
}

async function postCommand(command: string) {
  console.log("trying to post transition: ", command);
  let headers: any = {
    "Content-Type": "application/json",
  };
  headers[sourceReplicaIdKey] = document.myReplica;

  try {
    const response = await fetch(`/commands/${command}`, {
      method: "POST",
      mode: "same-origin",
      cache: "no-cache",
      headers,
      redirect: "follow",
      referrerPolicy: "no-referrer",
      body: "{}",
    });
    await response.json();
    const sourceReplicaId = response.headers.get(sourceReplicaIdKey);
    if (sourceReplicaId != document.myReplica) {
      addAlert(
        `Command sent to another replica: ${sourceReplicaId}!=${document.myReplica}.
        The state machine missed the command...`,
        "info",
      );
    }
  } catch (err) {
    console.log("ERROR: posting command:", err?.message ?? err);
  }
}

async function reRenderGraph(selectedState, progress) {
  let input = updateGraphDefinition(selectedState, progress);
  if (input === document.lastInput) {
    console.log("nothing to re-render");
    return;
  }
  document.lastInput = input;
  let rendered = await mermaid.mermaidAPI.render("temporary-graph", input);
  let graph = document.querySelector("#graph");
  if (graph) {
    graph.innerHTML = rendered.svg;
    bindGraphClicks();
  } else {
    console.log("ERROR: could not find target element for redrawing");
  }
}

function updateGraphDefinition(selectedState, progress) {
  let res = `stateDiagram-v2
  [*] --> waiting
  waiting --> working : start
  working --> aborting : abort
  working --> waiting
  aborting --> waiting
  classDef inProgress font-style:italic, stroke-dasharray: 5 5, stroke-width:3px;
  class ${selectedState} inProgress
  `;
  if (progress && progress.trim() !== "") {
    res += `note right of working
        ${progress}
    end note`;
  }
  return res;
}

function formatEventIntoOneLine(event) {
  let res = `${event.timestamp}: ${event.name}`;
  if (Object.keys(event?.properties ?? {}).length !== 0) {
    // res+=` ${Object.entries(event.properties)})`;
    res += ` [${Object.entries(event.properties)
      .map((e) => e[0] + ": " + e[1])
      .join(", ")}]`;
  }
  return res;
}

function hideDisconnectedAlert() {
  $("#offline-alert").hide();
}

function showDisconnectedAlert() {
  $("#offline-alert").show();
}

function flashConnectedAlert() {
  $("#connected-alert").show();
  $("#connected-alert").fadeTo(500, 50, function () {
    $("#connected-alert").slideUp(500);
  });
}

function addAlert(text: string, alertType: string) {
  // https://getbootstrap.com/docs/5.3/components/alerts/
  const alertPlaceholder = document.getElementById("alert-placeholder");
  const wrapper = document.createElement("div");
  wrapper.innerHTML = [
    `<div class="alert alert-${alertType} alert-dismissible fade show" role="alert">`,
    `   <div>${text}</div>`,
    '   <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>',
    "</div>",
  ].join("");
  setTimeout(function () {
    $(wrapper).find(".alert").alert("close");
  }, 3000);

  alertPlaceholder?.append(wrapper);
}
