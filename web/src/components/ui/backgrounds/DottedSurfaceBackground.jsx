import React, { useEffect, useRef } from 'react';
import * as THREE from 'three';

const DottedSurfaceBackground = ({ isDark }) => {
  const containerRef = useRef(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
    camera.position.set(0, 12, 25);
    camera.lookAt(0, -5, 0);

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
    renderer.setSize(window.innerWidth, window.innerHeight);
    container.appendChild(renderer.domElement);

    const particleCount = 10000;
    const geometry = new THREE.BufferGeometry();
    const positions = new Float32Array(particleCount * 3);
    const scales = new Float32Array(particleCount);

    const gridSize = 100;
    const spacing = 0.5;
    const offset = (gridSize * spacing) / 2;

    let i = 0;
    for (let ix = 0; ix < gridSize; ix++) {
      for (let iy = 0; iy < gridSize; iy++) {
        positions[i * 3] = ix * spacing - offset;
        positions[i * 3 + 1] = 0;
        positions[i * 3 + 2] = iy * spacing - offset;
        scales[i] = 1;
        i++;
      }
    }

    geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geometry.setAttribute('scale', new THREE.BufferAttribute(scales, 1));

    const material = new THREE.ShaderMaterial({
      uniforms: {
        color: { value: new THREE.Color(isDark ? 0xffffff : 0x000000) },
        time: { value: 0 }
      },
      vertexShader: `
        attribute float scale;
        uniform float time;
        varying float vDepth;
        void main() {
          vec3 p = position;
          p.y = sin(p.x * 0.2 + time) * 3.0 + cos(p.z * 0.2 + time) * 3.0;
          vec4 mvPosition = modelViewMatrix * vec4(p, 1.0);
          vDepth = -mvPosition.z;
          gl_PointSize = scale * (40.0 / vDepth);
          gl_Position = projectionMatrix * mvPosition;
        }
      `,
      fragmentShader: `
        uniform vec3 color;
        varying float vDepth;
        void main() {
          if (length(gl_PointCoord - vec2(0.5, 0.5)) > 0.475) discard;
          float fogFactor = smoothstep(15.0, 60.0, vDepth);
          float alpha = 1.0 * (1.0 - fogFactor);
          gl_FragColor = vec4(color, alpha);
        }
      `,
      transparent: true,
      depthWrite: false
    });

    const particles = new THREE.Points(geometry, material);
    scene.add(particles);

    let frameId;
    let time = 0;
    const animate = () => {
      time += 0.02;
      material.uniforms.time.value = time;
      
      particles.rotation.y = time * 0.05;
      
      renderer.render(scene, camera);
      frameId = requestAnimationFrame(animate);
    };
    animate();

    const handleResize = () => {
      camera.aspect = window.innerWidth / window.innerHeight;
      camera.updateProjectionMatrix();
      renderer.setSize(window.innerWidth, window.innerHeight);
    };
    window.addEventListener('resize', handleResize);

    return () => {
      cancelAnimationFrame(frameId);
      window.removeEventListener('resize', handleResize);
      if (container.contains(renderer.domElement)) {
        container.removeChild(renderer.domElement);
      }
      geometry.dispose();
      material.dispose();
      renderer.dispose();
    };
  }, [isDark]);

  return (
    <div 
      ref={containerRef} 
      className="absolute inset-0 pointer-events-none z-0"
      style={{ overflow: 'hidden' }}
    />
  );
};

export default DottedSurfaceBackground;
