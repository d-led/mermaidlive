import { getObjects } from "./src/client";
import { CatalogObject } from "./src/types";

console.log(`loaded index.js`);

$(async function () {
  await mermaid.run({
    querySelector: '.mermaid',
  });

  $(".node").on("click", function (e) {
    postCommand($(this).find(".nodeLabel:first").text());
  });

  console.log("ready");
  try {
    const objects = await getObjects();
    fillObjects(objects);
  } catch (e) {
    showText(`sorry, ${e.message ?? "an error happened"}`);
  }

  console.log("done");

  while(true) {
    console.log("subscribing");
    try {
      await subscribe();
    } catch (err) {
      console.log("ERROR:", err?.message ?? err)
    }
    console.log("waiting before reconnecting...")
    await sleep(5);
  }
});

async function subscribe() {
  let response = await fetch("/events");

  if (response.status == 502) {
    // reconnect on timeout
    await subscribe();
  } else if (response.status != 200) {
    // errored!
    console.log("ERROR:", response.statusText);
    // reconnect
    await sleep(3);
    await subscribe();
  } else {
    // Get and show the message
    const reader = response?.body?.getReader();
    if (!reader) {
      console.log("ERROR: failed to read the messages");
    } else {
      // read all messages
      let currentMessage="";
      while (true) {
        let chunk = await reader.read();
        if (chunk.done) {
          break;
        }
        currentMessage+=new TextDecoder('utf-8').decode(chunk.value);
        let endlineAt = currentMessage.indexOf('\n');
        if (endlineAt===-1) {
          console.log("incomplete chunk:", currentMessage)
          continue;
        }
        let messageToProcess = currentMessage.substring(0, endlineAt);
        currentMessage = currentMessage.substring(endlineAt+1);
        try {
          let message=JSON.parse(messageToProcess)
          if (message.name) {
            console.log("MESSAGE:", message);
            await processEvent(message)
          } else {
            showServerTime(message.timestamp);
          }
        } catch(err) {
            console.log("MESSAGE WAS:", currentMessage);
            console.log("ERROR:",err?.message ?? err)
        }
      }
    }
    // Call subscribe() again to try to reconnect
    await sleep(1);
    await subscribe();
  }
}

function fillObjects(objects: CatalogObject[]) {
  const objectsEl = $("#objects");

  objects.forEach((o, _) => {
    objectsEl.append(`
              <tr>
                  <th scope="row">${o.id}</th>
                  <td>${o.name}</td>
              </tr>
          `);
  });

  $("#objects-table").show();
}

function showText(text: string) {
  $("#delayed-text").text(text);
}

function showServerTime(text: string) {
  $("#server-time").text(text);
}

async function sleep(seconds: number) {
  await new Promise((resolve) => setTimeout(resolve, seconds * 1000 /*ms*/));
}

async function processEvent(event) {
  $("#last-event").text(`${event.timestamp}: ${event.name} (${JSON.stringify(event.properties)})`);
}

async function postCommand(command:string) {
  console.log("trying to post transition: ", command);
  try {
    const response = await fetch(`/commands/${command}`, {
      method: "POST",
      mode: "same-origin",
      cache: "no-cache",
      headers: {
        "Content-Type": "application/json",
      },
      redirect: "follow",
      referrerPolicy: "no-referrer",
      body: "{}",
    });
    console.log(await response.json());
  } catch (err) {
    console.log("ERROR: posting command:", err?.message ?? err);
  }
}
