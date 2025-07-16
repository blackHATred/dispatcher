import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls';

export default class PointCloud {
    constructor(containerId, sseUrl) {
        this.container = document.getElementById(containerId);
        this.sseUrl = sseUrl;
        this.initThree();
        this.initSSE();
    }

    initThree() {
        this.scene = new THREE.Scene();
        this.camera = new THREE.PerspectiveCamera(75, this.container.clientWidth / this.container.clientHeight, 0.1, 1000);
        this.camera.position.z = 50;
        this.renderer = new THREE.WebGLRenderer();
        this.renderer.setSize(this.container.clientWidth, this.container.clientHeight);
        this.container.appendChild(this.renderer.domElement);
        this.controls = new OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.screenSpacePanning = false;
        this.controls.minDistance = 1;
        this.controls.maxDistance = 500;
        this.points = null;
        this.animate();
    }

    initSSE() {
        const connect = () => {
            this.eventSource = new EventSource(this.sseUrl);
            this.eventSource.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.updatePointCloud(data);
            };
            this.eventSource.onerror = () => {
                this.eventSource.close();
                setTimeout(connect, 1000); // попытка переподключения через 1 сек
            };
        };
        connect();
    }

    updatePointCloud(pointsArray) {
        if (this.points) {
            this.scene.remove(this.points);
        }
        const geometry = new THREE.BufferGeometry();
        const vertices = new Float32Array(pointsArray.flat());
        geometry.setAttribute('position', new THREE.BufferAttribute(vertices, 3));

        // Градиент цвета по расстоянию
        const colors = [];
        let avgAlpha = 0;
        for (let i = 0; i < pointsArray.length; i++) {
            const [x, y, z] = pointsArray[i];
            const dist = Math.sqrt(x * x + y * y + z * z);
            // Цвет: ближе к центру - синий, дальше - красный
            const t = Math.min(dist / 500, 1);
            const r = t;
            const g = 0.2 * (1 - t);
            const b = 1 - t;
            colors.push(r, g, b);
            avgAlpha += 0.3 + 0.7 * (1 - t);
        }
        geometry.setAttribute('color', new THREE.BufferAttribute(new Float32Array(colors), 3));
        avgAlpha /= pointsArray.length;

        // Материал с поддержкой прозрачности, без текстуры (будут квадраты)
        const material = new THREE.PointsMaterial({
            size: 0.01,
            vertexColors: true,
            alphaTest: 0.01,
            transparent: true,
            opacity: avgAlpha
        });

        this.points = new THREE.Points(geometry, material);
        this.scene.add(this.points);
    }

    setCameraPreset(preset) {
        if (!this.camera || !this.controls) return;
        switch (preset) {
            case 'top':
                // Вид сверху: камера над облаком, смотрит вниз
                this.camera.position.set(0, 0, 50);
                this.camera.up.set(0, 1, 0);
                this.controls.target.set(0, 0, 0);
                this.camera.lookAt(0, 0, 0);
                break;
            case 'driver':
                // Вид от водителя: камера спереди и чуть выше центра, смотрит назад
                this.camera.position.set(0, -1, 0);
                this.camera.up.set(0, -1, 0);
                this.controls.target.set(0, 0, 0);
                this.camera.lookAt(0, 0, 0);
                break;
            case 'side':
                // Вид сбоку: камера слева от облака, смотрит на центр
                this.camera.position.set(-20, 2, 0);
                this.camera.up.set(0, 0, 1);
                this.controls.target.set(0, 0, 0);
                this.camera.lookAt(0, 0, 0);
                break;
            default:
                break;
        }
        this.controls.update();
    }

    animate() {
        requestAnimationFrame(() => this.animate());
        this.controls.update();
        this.renderer.render(this.scene, this.camera);
    }
}
