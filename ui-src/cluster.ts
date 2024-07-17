console.log(`loaded cluster.js`);

var lastInput = "";
var clusterEvents: { from: string; to: string; arrowText: string; }[] = [];

const sourceReplicaIdKey = "Source-Replica-Id";
const doCorrectMissingIdentities = true;

$(async function () {
  await reRenderGraph();

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
  await subscribe("/cluster/events", processEvent);
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

async function sleep(seconds: number) {
  await new Promise((resolve) => setTimeout(resolve, seconds * 1000 /*ms*/));
}

async function processEvent(event) {
  if (!event.name) {
    return;
  }

  switch (event.name) {
    case "StartedListening":
    case "ConnectedToRegion":
      //ignore
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
        break;
    case "TotalVisitors":
      showTotalVisitors(event?.properties?.param);
      // do not show this event in the log
      return;
    case "ClusterMessage":
      processClusterMessage(event);
      await reRenderGraph();
      // do not show this event in the log
      return;
    default:
      console.log(`unhandled event: ${event.name}`);
      return;
  }

  console.log("INCOMING_EVENT:", event);
}

async function reRenderGraph() {
  if (!clusterEvents?.length) {
    console.log("nothing to render");
    return;
  }
  let input = updateGraphDefinition();
  if (input === lastInput) {
    console.log("nothing to re-render");
    return;
  }
  lastInput = input;
  try {
    let rendered = await mermaid.mermaidAPI.render("temporary-graph", input);
    let graph = document.querySelector("#graph");
    if (graph) {
      graph.innerHTML = rendered.svg;
    } else {
      console.log("ERROR: could not find target element for redrawing");
    }
  } catch (e) {
    console.log("error rendering graph:", e);
  }
}

function processClusterMessage(event) {
  const e = renderableClusterEvent(event);
  if (!e.arrowText) {
    return;
  }
  clusterEvents.push(e);
  if (doCorrectMissingIdentities && (e.arrowText.indexOf('hello')==0 || e.arrowText.indexOf('ohai')==0)) {
    correctClusterEvents(e);
  }
}

function renderableClusterEvent(event) {
  let arrowText = arrowTextFrom(event);
  let from = normalizeParticipant(event?.properties?.src ?? 'unknown-src');
  let to = normalizeParticipant(event?.properties?.dst ?? 'unknown-dst');
  let mapping = ipMappingFrom(event);
  let re = {
    from,
    to,
    arrowText,
    mapping,
  };
  return re;
}

function correctClusterEvents(newEvent) {
  console.log("mapping", newEvent.mapping);
  if (!newEvent.mapping) {
    return;
  }
  clusterEvents.forEach((e,i) =>{
    if (`${e.from}`.indexOf(newEvent.mapping.ip) !== -1) {
      e.from = newEvent.mapping.peer;
      console.log(`${e.from} -> ${newEvent.mapping.peer}`);
      clusterEvents[i] = e;
    }
    if (`${e.to}`.indexOf(newEvent.mapping.ip) !== -1) {
      e.to = newEvent.mapping.peer;
      console.log(`${e.to} -> ${newEvent.mapping.peer}`);
      clusterEvents[i] = e;
    }
  });
}

function normalizeParticipant(participant) {
  if (participant.indexOf(':') == -1) {
    return participant;
  }
  
  // opportunistic: ipv6
  let url = tryParseUrl(participant);
  if (!!url) {
    return url;
  }
  // opportunistic: ipv4
  url = tryParseUrl(participant.replaceAll('[','').replaceAll(']', ''));
  if (!!url) {
    return url;
  }
  return `${participant}`.replaceAll(':', '/');
}

function tryParseUrl(input) {
  try {
    const url = new URL(input);
    return url.hostname;
  } catch (e) {
    return null;
  }
}

function arrowTextFrom(event) {
  try {
    let orig = JSON.parse(event?.properties?.msg);
    if (orig.name && orig.peers) {
      return JSON.stringify({
        name: orig.name,
        peers: orig.peers,
        total: sumUp(orig.peers),
      });
    } else {
      switch (orig?.type) {
        case "peer.ohai.network.message":
          return `ohai, ${orig?.metadata?.my_ip} == ${orig?.source_peer}`;
        case "peer.hello.network.message":
          return `hello, ${orig?.metadata?.my_ip} == ${orig?.source_peer}`;
        default:
          return event?.properties?.msg ?? 'unknown-msg';
      }
    }
  } catch (e) {
    console.log("error parsing message", e)
  }
}

function ipMappingFrom(event) {
  try {
    const orig = JSON.parse(event?.properties?.msg);
    const ip = orig?.metadata?.my_ip;
    const peer = orig?.source_peer;
    if (ip && peer) {
        return ({ip, peer})
    }
  } catch (e) {
    console.log("error parsing message", e)
  }
  return null;
}

function sumUp(peers) {
  let sum = 0;
  Object.keys(peers).forEach(peer=>sum+=parseInt(peers[peer]));
  return sum;
}

function updateGraphDefinition() {
  let res = `sequenceDiagram
  `;
  clusterEvents.forEach(e => {
    res+=`${e.from}->>+${e.to}: ${e.arrowText}
    `
  });
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
  const alertPlaceholder = document.getElementById('alert-placeholder');
    const wrapper = document.createElement('div')
    wrapper.innerHTML = [
      `<div class="alert alert-${alertType} alert-dismissible fade show" role="alert">`,
      `   <div>${text}</div>`,
      '   <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>',
      '</div>'
    ].join('')
    setTimeout(function() {
      $(wrapper).find('.alert').alert('close');
    }, 3000);
  
    alertPlaceholder?.append(wrapper)
}
