import './style.css'
import PointCloud from './PointCloud.js';

document.querySelector('#app').innerHTML = `
  <div>
    <h1>Визуализация облака точек</h1>
    <div style="margin: 10px 0;">
      <button id="preset-top">Вид сверху</button>
      <button id="preset-driver">Вид от водителя</button>
      <button id="preset-side">Вид сбоку</button>
    </div>
    <div id="pointcloud-container" style="width: 80vw; height: 80vh; border: 1px solid #333; margin-top: 20px;"></div>
  </div>
`

// Инициализация облака точек
let pointCloudInstance;
window.addEventListener('DOMContentLoaded', () => {
  pointCloudInstance = new PointCloud('pointcloud-container', `${window.location.origin}/sse`);
  document.getElementById('preset-top').onclick = () => pointCloudInstance.setCameraPreset('top');
  document.getElementById('preset-driver').onclick = () => pointCloudInstance.setCameraPreset('driver');
  document.getElementById('preset-side').onclick = () => pointCloudInstance.setCameraPreset('side');
});
