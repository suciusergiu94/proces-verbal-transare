import './style.css';
import { Greet } from '../wailsjs/go/main/App';

document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
  <div>
    <h1>proces-verbal-transare</h1>
    <div id="result">Please enter your name below 👇</div>
    <div class="input-box">
      <input id="name" class="input" type="text" autocomplete="off" />
      <button class="btn" id="greet">Greet</button>
    </div>
  </div>
`;

const resultElement = document.getElementById('result') as HTMLDivElement;
const nameElement = document.getElementById('name') as HTMLInputElement;

function greet() {
    Greet(nameElement.value).then((result) => {
        resultElement.innerText = result;
    });
}

document.getElementById('greet')!.addEventListener('click', greet);
