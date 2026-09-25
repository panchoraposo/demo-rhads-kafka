const statusEl = document.getElementById('status');
const actions = document.getElementById('actions');
let current = null;

document.getElementById('planForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const body = {
    destination: fd.get('destination'),
    startDate: fd.get('startDate'),
    days: Number(fd.get('days')),
    tripType: fd.get('tripType'),
    travelers: Number(fd.get('travelers')),
    budget: Number(fd.get('budget')),
    preferences: fd.get('preferences'),
  };
  statusEl.textContent = 'Planning…';
  actions.style.display = 'none';
  const res = await fetch('/trip/plan', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  current = await res.json();
  render();
  if (current.status === 'awaiting_approval') actions.style.display = 'block';
});

async function decide(status) {
  if (!current?.instanceId) return;
  statusEl.textContent = 'Submitting decision…';
  await fetch('/trip/approve', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ instanceId: current.instanceId, status }),
  });
  for (let i = 0; i < 40; i++) {
    await new Promise((r) => setTimeout(r, 500));
    const res = await fetch('/trip/plan/status?instanceId=' + encodeURIComponent(current.instanceId));
    current = await res.json();
    render();
    if (['confirmed', 'rejected', 'failed'].includes(current.status)) break;
  }
}

document.getElementById('approve').onclick = () => decide('approved');
document.getElementById('reject').onclick = () => decide('rejected');

function render() {
  statusEl.textContent = JSON.stringify(current, null, 2);
}

(async () => {
  const res = await fetch('/trip/plan/latest');
  if (res.status === 200) {
    current = await res.json();
    render();
    if (current.status === 'awaiting_approval') actions.style.display = 'block';
  }
})();
